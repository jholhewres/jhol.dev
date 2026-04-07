import { createContext, useContext, useState, useEffect, ReactNode } from "react";

type Mode = "system" | "light" | "dark";
type Resolved = "light" | "dark";

interface ThemeContextType {
  mode: Mode;
  resolved: Resolved;
  setMode: (m: Mode) => void;
  cycle: () => void; // system -> light -> dark -> system
}

const ThemeContext = createContext<ThemeContextType | null>(null);

function getSystemPref(): Resolved {
  if (typeof window === "undefined") return "light";
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function loadMode(): Mode {
  if (typeof localStorage === "undefined") return "system";
  const stored = localStorage.getItem("theme");
  if (stored === "light" || stored === "dark" || stored === "system") return stored;
  return "system";
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [mode, setModeState] = useState<Mode>(loadMode);
  const [resolved, setResolved] = useState<Resolved>(() =>
    mode === "system" ? getSystemPref() : mode
  );

  // Apply theme to <html>
  useEffect(() => {
    const r = mode === "system" ? getSystemPref() : mode;
    setResolved(r);
    document.documentElement.setAttribute("data-theme", r);
  }, [mode]);

  // Listen for system pref changes when in system mode
  useEffect(() => {
    if (mode !== "system") return;
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const handler = (e: MediaQueryListEvent) => {
      const r: Resolved = e.matches ? "dark" : "light";
      setResolved(r);
      document.documentElement.setAttribute("data-theme", r);
    };
    mq.addEventListener("change", handler);
    return () => mq.removeEventListener("change", handler);
  }, [mode]);

  const setMode = (m: Mode) => {
    setModeState(m);
    localStorage.setItem("theme", m);
  };

  const cycle = () => {
    const next: Mode = mode === "system" ? "light" : mode === "light" ? "dark" : "system";
    setMode(next);
  };

  return (
    <ThemeContext.Provider value={{ mode, resolved, setMode, cycle }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme() {
  const ctx = useContext(ThemeContext);
  if (!ctx) throw new Error("useTheme must be used within ThemeProvider");
  return ctx;
}
