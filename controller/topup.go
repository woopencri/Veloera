// Copyright (c) 2025 Tethys Plex
//
// This file is part of Veloera.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.
package controller

import (
	"crypto/md5"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"veloera/common"
	"veloera/model"
	"veloera/service"
	"veloera/setting"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type EpayRequest struct {
	Amount        int64  `json:"amount"`
	PaymentMethod string `json:"payment_method"`
	TopUpCode     string `json:"top_up_code"`
}

type AmountRequest struct {
	Amount    int64  `json:"amount"`
	TopUpCode string `json:"top_up_code"`
}

// generateSign generates a signature string based on the provided parameters and key.
// This function is specifically designed to be compatible with the "码支付AliMPay" project's
// private signature algorithm.
func generateSign(params map[string]string, key string) string {
	// 1. Create a slice of keys
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	// 2. Sort the keys alphabetically
	sort.Strings(keys)

	// 3. Build the string to be signed, filtering out empty values
	var builder strings.Builder
	for _, k := range keys {
		if v := params[k]; v != "" {
			if builder.Len() > 0 {
				builder.WriteString("&")
			}
			builder.WriteString(k)
			builder.WriteString("=")
			builder.WriteString(v)
		}
	}

	// 4. Append the merchant key
	signStr := builder.String() + key

	// 5. Calculate MD5 hash
	return fmt.Sprintf("%x", md5.Sum([]byte(signStr)))
}

func getPayMoney(amount int64, group string) float64 {
	dAmount := decimal.NewFromInt(amount)

	if !common.DisplayInCurrencyEnabled {
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		dAmount = dAmount.Div(dQuotaPerUnit)
	}

	topupGroupRatio := common.GetTopupGroupRatio(group)
	if topupGroupRatio == 0 {
		topupGroupRatio = 1
	}

	dTopupGroupRatio := decimal.NewFromFloat(topupGroupRatio)
	dPrice := decimal.NewFromFloat(setting.Price)

	payMoney := dAmount.Mul(dPrice).Mul(dTopupGroupRatio)

	return payMoney.InexactFloat64()
}

func getMinTopup() int64 {
	minTopup := setting.MinTopUp
	if !common.DisplayInCurrencyEnabled {
		dMinTopup := decimal.NewFromInt(int64(minTopup))
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		minTopup = int(dMinTopup.Mul(dQuotaPerUnit).IntPart())
	}
	return int64(minTopup)
}

func RequestEpay(c *gin.Context) {
	var req EpayRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	if req.Amount < getMinTopup() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", getMinTopup())})
		return
	}

	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getPayMoney(req.Amount, group)
	if payMoney < 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}

	if setting.PayAddress == "" || setting.EpayId == "" || setting.EpayKey == "" {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "当前管理员未配置支付信息"})
		return
	}

	// Force payment type to alipay, as it's the only one supported by the server
	payType := "alipay"

	callBackAddress := service.GetCallbackAddress()
	tradeNo := fmt.Sprintf("USR%dNO%s", id, fmt.Sprintf("%s%d", common.GetRandomString(6), time.Now().Unix()))

	// Prepare parameters for signing, matching the "码支付AliMPay" protocol
	params := map[string]string{
		"pid":          setting.EpayId,
		"type":         payType,
		"out_trade_no": tradeNo,
		"notify_url":   fmt.Sprintf("%s/api/user/epay/notify", callBackAddress),
		"return_url":   fmt.Sprintf("%s/log", setting.ServerAddress),
		"name":         fmt.Sprintf("TUC%d", req.Amount),
		"money":        strconv.FormatFloat(payMoney, 'f', 2, 64),
		"sitename":     "Veloera",
	}

	// Generate the signature using the correct algorithm
	sign := generateSign(params, setting.EpayKey)

	// Add signature to the parameters to be sent
	params["sign"] = sign
	params["sign_type"] = "MD5"

	// Build the final URL for redirection
	baseURL, _ := url.Parse(setting.PayAddress)
	queryParams := url.Values{}
	for k, v := range params {
		queryParams.Add(k, v)
	}
	baseURL.RawQuery = queryParams.Encode()
	finalURL := baseURL.String()

	// Record the top-up request in the database
	topUp := &model.TopUp{
		UserId:  id,
		Amount:  req.Amount,
		Money:   payMoney,
		TradeNo: tradeNo,
		Status:  "pending",
	}
	err = topUp.Insert()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": err.Error()})
		return
	}

	// Return the URL for the frontend to redirect
	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data":    finalURL,
	})
}

var (
	notifyLock sync.Mutex
)

// EpayNotify handles the callback notification from the payment server.
func EpayNotify(c *gin.Context) {
	notifyLock.Lock()
	defer notifyLock.Unlock()

	// 1. Bind all query parameters into a map
	queryMap := make(map[string]string)
	for k, v := range c.Request.URL.Query() {
		if len(v) > 0 {
			queryMap[k] = v[0]
		}
	}

	// 2. Check for essential parameters
	if queryMap["trade_status"] == "" || queryMap["out_trade_no"] == "" {
		log.Println("epay notify: missing essential parameters")
		c.String(http.StatusBadRequest, "fail")
		return
	}

	// 3. Separate the signature from the parameters to be verified
	receivedSign, ok := queryMap["sign"]
	if !ok {
		log.Println("epay notify: sign parameter missing")
		c.String(http.StatusBadRequest, "fail")
		return
	}
	delete(queryMap, "sign")
	delete(queryMap, "sign_type") // sign_type is not included in server's signature calculation

	// 4. Verify the signature
	expectedSign := generateSign(queryMap, setting.EpayKey)
	if receivedSign != expectedSign {
		log.Printf("epay notify: signature mismatch. expected=%s, received=%s", expectedSign, receivedSign)
		c.String(http.StatusBadRequest, "fail")
		return
	}

	// 5. Process the order if signature is valid
	if queryMap["trade_status"] == "TRADE_SUCCESS" {
		tradeNo := queryMap["out_trade_no"]
		topUp, err := model.GetTopUpByTradeNo(tradeNo)
		if err != nil {
			log.Printf("get topup by trade no error: %v", err)
			c.String(http.StatusInternalServerError, "fail")
			return
		}
		if topUp.Status == "paid" {
			c.String(http.StatusOK, "success")
			return
		}
		topUp.Status = "paid"
		err = topUp.Update()
		if err != nil {
			log.Printf("update topup error: %v", err)
			c.String(http.StatusInternalServerError, "fail")
			return
		}
		err = model.IncreaseUserQuota(topUp.UserId, int(topUp.Amount))
		if err != nil {
			log.Printf("increase user quota error: %v", err)
			c.String(http.StatusInternalServerError, "fail")
			return
		}
		model.RecordLog(topUp.UserId, model.LogTypeTopup, fmt.Sprintf("使用在线充值成功，充值金额: %d", topUp.Amount))
	}
	c.String(http.StatusOK, "success")
}

func GetTopup(c *gin.Context) {
	userId := c.GetInt("id")
	p, _ := strconv.Atoi(c.Query("p"))
	if p < 1 {
		p = 1
	}
	topups, err := model.GetTopUpsByUserId(userId, (p-1)*common.ItemsPerPage, common.ItemsPerPage)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    topups,
	})
}

func GetMinAmount(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    getMinTopup(),
	})
}

func GetAmount(c *gin.Context) {
	var req AmountRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getPayMoney(req.Amount, group)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    payMoney,
	})
}
