import { Link } from "react-router-dom";
import ThemeToggle from "./ThemeToggle";

export default function BackNav() {
  return (
    <div className="flex items-center justify-between mb-6">
      <Link
        to="/"
        className="inline-block text-sm text-[var(--color-fg-subtle)] hover:text-accent transition-colors"
      >
        &larr; jhol.dev
      </Link>
      <ThemeToggle />
    </div>
  );
}
