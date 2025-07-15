'use client';

import { useState } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { 
  Home, 
  Users, 
  Database, 
  BarChart, 
  MessageSquare, 
  Settings,
  CreditCard,
  Layers,
  Key,
  ChevronDown,
  ChevronRight
} from 'lucide-react';
import { useI18n } from '@/i18n';
import { useAuth } from '@/contexts/auth-context';
import { cn } from '@/lib/utils';

interface SidebarItemProps {
  href: string;
  icon: React.ReactNode;
  text: string;
  isActive: boolean;
  hasSubItems?: boolean;
  isExpanded?: boolean;
  onClick?: () => void;
}

const SidebarItem = ({ 
  href, 
  icon, 
  text, 
  isActive, 
  hasSubItems = false,
  isExpanded = false,
  onClick 
}: SidebarItemProps) => {
  return (
    <Link
      href={href}
      className={cn(
        "flex items-center px-4 py-2 text-sm rounded-md mb-1",
        isActive 
          ? "bg-sidebar-primary text-sidebar-primary-foreground" 
          : "text-sidebar-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
      )}
      onClick={onClick}
    >
      <span className="mr-3">{icon}</span>
      <span className="flex-1">{text}</span>
      {hasSubItems && (
        <span className="ml-2">
          {isExpanded ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
        </span>
      )}
    </Link>
  );
};

interface SidebarGroupProps {
  title: string;
  icon: React.ReactNode;
  items: {
    href: string;
    text: string;
    icon: React.ReactNode;
  }[];
  defaultOpen?: boolean;
}

const SidebarGroup = ({ title, icon, items, defaultOpen = false }: SidebarGroupProps) => {
  const [isOpen, setIsOpen] = useState(defaultOpen);
  const pathname = usePathname();
  const isActive = items.some(item => pathname === item.href);
  
  return (
    <div className="mb-2">
      <button
        className={cn(
          "flex items-center w-full px-4 py-2 text-sm rounded-md mb-1",
          isActive 
            ? "bg-sidebar-primary text-sidebar-primary-foreground" 
            : "text-sidebar-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
        )}
        onClick={() => setIsOpen(!isOpen)}
      >
        <span className="mr-3">{icon}</span>
        <span className="flex-1">{title}</span>
        <span className="ml-2">
          {isOpen ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
        </span>
      </button>
      
      {isOpen && (
        <div className="ml-6 mt-1 space-y-1">
          {items.map((item, index) => (
            <Link
              key={index}
              href={item.href}
              className={cn(
                "flex items-center px-4 py-2 text-sm rounded-md",
                pathname === item.href
                  ? "bg-sidebar-primary text-sidebar-primary-foreground"
                  : "text-sidebar-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
              )}
            >
              <span className="mr-3">{item.icon}</span>
              <span>{item.text}</span>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
};

export default function Sidebar() {
  const { t } = useI18n();
  const { user, isAuthenticated } = useAuth();
  const pathname = usePathname();
  
  // Only show sidebar for authenticated users
  if (!isAuthenticated) {
    return null;
  }
  
  // Check if user is admin (role 100)
  const isAdmin = user?.role === 100;
  
  return (
    <aside className="w-64 bg-sidebar text-sidebar-foreground border-r border-sidebar-border h-screen overflow-y-auto fixed left-0 top-16 pt-5 pb-20">
      <div className="px-3">
        <SidebarItem
          href="/"
          icon={<Home size={20} />}
          text={t('common.home')}
          isActive={pathname === '/'}
        />
        
        <SidebarItem
          href="/profile"
          icon={<Users size={20} />}
          text={t('user.profile')}
          isActive={pathname === '/profile'}
        />
        
        {isAdmin && (
          <SidebarGroup
            title={t('channel.title')}
            icon={<Database size={20} />}
            items={[
              {
                href: '/admin/channels',
                text: t('channel.list'),
                icon: <Database size={16} />
              },
              {
                href: '/channel/create',
                text: t('channel.create'),
                icon: <Database size={16} />
              }
            ]}
            defaultOpen={pathname.startsWith('/admin/channels')}
          />
        )}
        
        <SidebarGroup
          title={t('token.title')}
          icon={<Key size={20} />}
          items={[
            {
              href: '/token',
              text: t('token.list'),
              icon: <Key size={16} />
            },
            {
              href: '/token/create',
              text: t('token.create'),
              icon: <Key size={16} />
            }
          ]}
          defaultOpen={pathname.startsWith('/token')}
        />
        
        <SidebarGroup
          title={t('log.title')}
          icon={<Layers size={20} />}
          items={[
            {
              href: '/log/usage',
              text: t('log.usage'),
              icon: <Layers size={16} />
            },
            {
              href: '/log/midjourney',
              text: t('log.midjourney'),
              icon: <Layers size={16} />
            },
            {
              href: '/log/task',
              text: t('log.task'),
              icon: <Layers size={16} />
            }
          ]}
          defaultOpen={pathname.startsWith('/log')}
        />
        
        <SidebarItem
          href="/dashboard"
          icon={<BarChart size={20} />}
          text={t('dashboard.title')}
          isActive={pathname === '/dashboard'}
        />
        
        <SidebarGroup
          title={t('chat.title')}
          icon={<MessageSquare size={20} />}
          items={[
            {
              href: '/chat',
              text: t('chat.interface'),
              icon: <MessageSquare size={16} />
            },
            {
              href: '/chat/chat2link',
              text: t('chat.chat2link'),
              icon: <MessageSquare size={16} />
            },
            {
              href: '/chat/playground',
              text: t('chat.playground'),
              icon: <MessageSquare size={16} />
            }
          ]}
          defaultOpen={pathname.startsWith('/chat')}
        />
        
        {isAdmin && (
          <SidebarGroup
            title={t('system.title')}
            icon={<Settings size={20} />}
            items={[
              {
                href: '/system/operation',
                text: t('system.operation'),
                icon: <Settings size={16} />
              },
              {
                href: '/system/settings',
                text: t('system.settings'),
                icon: <Settings size={16} />
              },
              {
                href: '/system/other',
                text: t('system.other'),
                icon: <Settings size={16} />
              }
            ]}
            defaultOpen={pathname.startsWith('/system')}
          />
        )}
        
        <SidebarGroup
          title={t('pricing.title')}
          icon={<CreditCard size={20} />}
          items={[
            {
              href: '/pricing',
              text: t('pricing.plans'),
              icon: <CreditCard size={16} />
            },
            {
              href: '/pricing/redemption',
              text: t('pricing.redemption.title'),
              icon: <CreditCard size={16} />
            },
            {
              href: '/pricing/topup',
              text: t('pricing.topup.title'),
              icon: <CreditCard size={16} />
            }
          ]}
          defaultOpen={pathname.startsWith('/pricing')}
        />
      </div>
    </aside>
  );
}
