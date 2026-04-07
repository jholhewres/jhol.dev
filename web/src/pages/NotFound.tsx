import { Link } from "react-router-dom";

export default function NotFound() {
  return (
    <div className="py-20 text-center animate-fade-in">
      <h1 className="text-3xl font-bold text-[var(--color-fg)] mb-3">404</h1>
      <p className="text-[var(--color-fg-muted)] mb-6">Page not found.</p>
      <Link to="/" className="text-accent hover:text-accent-hover transition-colors">
        ← Back to home
      </Link>
    </div>
  );
}
