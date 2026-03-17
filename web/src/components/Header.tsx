import { useState } from "react";
import { Link, useLocation } from "react-router-dom";
import { useLanguage } from "../context/LanguageContext";

const navItems = [
  { key: "nav.about", path: "/about" },
  { key: "nav.blog", path: "/blog" },
  { key: "nav.projects", path: "/projects" },
  { key: "nav.experience", path: "/experience" },
  { key: "nav.contact", path: "/contact" },
];

export default function Header() {
  const { lang, setLang, t } = useLanguage();
  const location = useLocation();
  const [menuOpen, setMenuOpen] = useState(false);

  return (
    <header className="border-b border-gray-100">
      <div className="mx-auto max-w-[680px] px-5 py-4 flex items-center justify-between">
        <Link
          to="/"
          className="text-lg font-semibold text-[#1a1a1a] hover:text-accent transition-colors"
        >
          jhol.dev
        </Link>

        {/* Desktop nav */}
        <nav className="hidden md:flex items-center gap-6">
          {navItems.map((item) => (
            <Link
              key={item.path}
              to={item.path}
              className={`text-sm transition-colors ${
                location.pathname === item.path
                  ? "text-accent font-medium"
                  : "text-gray-600 hover:text-[#1a1a1a]"
              }`}
            >
              {t(item.key)}
            </Link>
          ))}
          <button
            onClick={() => setLang(lang === "en" ? "pt" : "en")}
            className="text-sm text-gray-500 hover:text-[#1a1a1a] transition-colors border border-gray-200 rounded px-2 py-0.5"
          >
            {lang === "en" ? "PT" : "EN"}
          </button>
        </nav>

        {/* Mobile hamburger */}
        <button
          className="md:hidden p-1"
          onClick={() => setMenuOpen(!menuOpen)}
          aria-label="Toggle menu"
        >
          <svg
            className="w-6 h-6"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            {menuOpen ? (
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M6 18L18 6M6 6l12 12"
              />
            ) : (
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M4 6h16M4 12h16M4 18h16"
              />
            )}
          </svg>
        </button>
      </div>

      {/* Mobile menu */}
      {menuOpen && (
        <nav className="md:hidden border-t border-gray-100 px-5 py-3 space-y-2">
          {navItems.map((item) => (
            <Link
              key={item.path}
              to={item.path}
              onClick={() => setMenuOpen(false)}
              className={`block text-sm py-1 ${
                location.pathname === item.path
                  ? "text-accent font-medium"
                  : "text-gray-600"
              }`}
            >
              {t(item.key)}
            </Link>
          ))}
          <button
            onClick={() => {
              setLang(lang === "en" ? "pt" : "en");
              setMenuOpen(false);
            }}
            className="text-sm text-gray-500 border border-gray-200 rounded px-2 py-0.5 mt-1"
          >
            {lang === "en" ? "PT" : "EN"}
          </button>
        </nav>
      )}
    </header>
  );
}
