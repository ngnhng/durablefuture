import { NavLink } from "react-router";
import { useState } from "react";

// Simple SVG icons
const ChevronDownIcon = ({ className }: { className?: string }) => (
  <svg
    className={className}
    fill="none"
    stroke="currentColor"
    viewBox="0 0 24 24"
  >
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M19 9l-7 7-7-7"
    />
  </svg>
);

const ChevronRightIcon = ({ className }: { className?: string }) => (
  <svg
    className={className}
    fill="none"
    stroke="currentColor"
    viewBox="0 0 24 24"
  >
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M9 5l7 7-7 7"
    />
  </svg>
);

interface NavigationItem {
  name: string;
  href?: string;
  icon?: React.ComponentType<{ className?: string }>;
  children?: NavigationItem[];
}

const navigation: NavigationItem[] = [
  {
    name: "Environment",
    href: "/environment",
  },
  {
    name: "Development",
    children: [
      { name: "Tasks", href: "/tasks" },
      { name: "Runs", href: "/runs" },
      { name: "Batches", href: "/batches" },
      { name: "Schedules", href: "/schedules" },
      { name: "Queues", href: "/queues" },
      { name: "Deployments", href: "/deployments" },
      { name: "Test", href: "/test" },
    ],
  },
  {
    name: "Wallpoints",
    children: [{ name: "Tokens", href: "/tokens" }],
  },
  {
    name: "Manage",
    children: [
      { name: "Bulk actions", href: "/bulk-actions" },
      { name: "API keys", href: "/api-keys" },
      { name: "Environment variables", href: "/environment-variables" },
      { name: "Alerts", href: "/alerts" },
      { name: "Preview branches", href: "/preview-branches" },
      { name: "Regions", href: "/regions" },
      { name: "Project settings", href: "/project-settings" },
    ],
  },
];

interface NavigationItemProps {
  item: NavigationItem;
  level?: number;
}

function NavigationItem({ item, level = 0 }: NavigationItemProps) {
  const [isOpen, setIsOpen] = useState(true);

  const hasChildren = item.children && item.children.length > 0;
  const paddingLeft = level === 0 ? "pl-4" : "pl-8";

  if (hasChildren) {
    return (
      <div>
        <button
          onClick={() => setIsOpen(!isOpen)}
          className={`w-full flex items-center justify-between ${paddingLeft} pr-4 py-2 text-sm text-gray-300 hover:bg-gray-800 hover:text-white`}
        >
          <span className="flex items-center gap-2">
            {item.icon && <item.icon className="h-4 w-4" />}
            {item.name}
          </span>
          {isOpen ? (
            <ChevronDownIcon className="h-4 w-4" />
          ) : (
            <ChevronRightIcon className="h-4 w-4" />
          )}
        </button>
        {isOpen && (
          <div className="bg-gray-900">
            {item.children?.map((child) => (
              <NavigationItem key={child.name} item={child} level={level + 1} />
            ))}
          </div>
        )}
      </div>
    );
  }

  if (item.href) {
    return (
      <NavLink
        to={item.href}
        className={({ isActive }) =>
          `block ${paddingLeft} pr-4 py-2 text-sm transition-colors ${
            isActive
              ? "bg-blue-600 text-white"
              : "text-gray-300 hover:bg-gray-800 hover:text-white"
          }`
        }
      >
        <span className="flex items-center gap-2">
          {item.icon && <item.icon className="h-4 w-4" />}
          {item.name}
        </span>
      </NavLink>
    );
  }

  return null;
}

export default function Sidebar() {
  return (
    <div className="w-64 bg-gray-900 border-r border-gray-700 flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b border-gray-700">
        <div className="flex items-center gap-2">
          <div className="w-6 h-6 bg-blue-600 rounded"></div>
          <span className="font-semibold text-white">test</span>
        </div>
        <button className="text-gray-400 hover:text-white">
          <ChevronDownIcon className="h-4 w-4" />
        </button>
      </div>

      {/* Navigation */}
      <nav className="flex-1 overflow-y-auto py-4">
        {navigation.map((item) => (
          <NavigationItem key={item.name} item={item} />
        ))}
      </nav>

      {/* Footer */}
      <div className="border-t border-gray-700 p-4">
        <div className="flex items-center justify-between text-sm">
          <span className="text-gray-400">Help & Feedback</span>
          <button className="text-gray-400 hover:text-white">
            <span className="text-xs">⌘</span>
          </button>
        </div>
        <div className="mt-2 text-xs text-gray-500">
          Free Plan
          <button className="ml-2 text-blue-400 hover:text-blue-300">
            Upgrade
          </button>
        </div>
      </div>
    </div>
  );
}

export { Sidebar };
