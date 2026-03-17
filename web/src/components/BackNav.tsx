import { Link } from "react-router-dom";

export default function BackNav() {
  return (
    <Link
      to="/"
      className="inline-block text-sm text-gray-400 hover:text-accent transition-colors mb-6"
    >
      &larr; jhol.dev
    </Link>
  );
}
