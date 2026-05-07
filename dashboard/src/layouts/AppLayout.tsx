import { Outlet, NavLink } from "react-router-dom";
import { useTheme } from "next-themes";
import { Moon, Sun, GitGraph } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { useAuth } from "@/auth/useAuth";

function ThemeToggle() {
  const { theme, setTheme } = useTheme();
  return (
    <Button
      variant="ghost"
      size="icon"
      onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
      aria-label="Toggle dark mode"
    >
      <Sun className="h-5 w-5 rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
      <Moon className="absolute h-5 w-5 rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
    </Button>
  );
}

function navClass({ isActive }: { isActive: boolean }) {
  return `flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-colors ${
    isActive
      ? "bg-accent text-accent-foreground"
      : "hover:bg-accent/50 text-muted-foreground hover:text-foreground"
  }`;
}

export function AppLayout() {
  const { user, logout } = useAuth();
  const initials = user?.email?.slice(0, 2).toUpperCase() ?? "?";

  return (
    <div className="flex min-h-screen">
      {/* Sidebar */}
      <aside className="hidden md:flex flex-col w-56 border-r bg-background px-3 py-4 gap-1">
        <div className="flex items-center gap-2 px-3 py-2 mb-4">
          <GitGraph className="h-5 w-5 text-primary" />
          <span className="font-semibold text-sm">Dago</span>
        </div>
        <NavLink to="/graphs" className={navClass}>
          <GitGraph className="h-4 w-4" />
          Graphs
        </NavLink>
        <span className="flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium text-muted-foreground/50 cursor-not-allowed">
          Executions
        </span>
        <span className="flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium text-muted-foreground/50 cursor-not-allowed">
          Catalog
        </span>
      </aside>

      {/* Main */}
      <div className="flex flex-col flex-1">
        <header className="flex items-center justify-between border-b px-6 h-14">
          <span className="font-semibold text-sm md:hidden">Dago</span>
          <div className="flex-1" />
          <div className="flex items-center gap-2">
            <ThemeToggle />
            <Button variant="ghost" size="icon" onClick={logout} aria-label="Sign out">
              <Avatar className="h-8 w-8">
                <AvatarFallback className="text-xs">{initials}</AvatarFallback>
              </Avatar>
            </Button>
          </div>
        </header>
        <main className="flex-1">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
