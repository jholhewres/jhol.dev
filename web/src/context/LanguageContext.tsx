import { createContext, useContext, useState, useEffect, ReactNode } from "react";

type Lang = "en" | "pt";

interface LanguageContextType {
  lang: Lang;
  setLang: (lang: Lang) => void;
  t: (key: string) => string;
}

const translations: Record<string, Record<Lang, string>> = {
  "nav.about": { en: "About", pt: "Sobre" },
  "nav.blog": { en: "Blog", pt: "Blog" },
  "nav.projects": { en: "Projects", pt: "Projetos" },
  "nav.experience": { en: "Experience", pt: "Experiencia" },
  "nav.contact": { en: "Contact", pt: "Contato" },
  "home.greeting": { en: "Hi, I'm", pt: "Ola, eu sou" },
  "home.headline": {
    en: "AI Engineer",
    pt: "Engenheiro de IA",
  },
  "home.subtitle": {
    en: "Building production-ready AI systems. Multi-agent architecture, RAG & LLMOps.",
    pt: "Construindo sistemas de IA para producao. Arquitetura multi-agentes, RAG & LLMOps.",
  },
  "home.recent_posts": { en: "Recent Posts", pt: "Posts Recentes" },
  "home.view_all": { en: "View all posts", pt: "Ver todos os posts" },
  "blog.title": { en: "Blog", pt: "Blog" },
  "blog.reading_time": { en: "min read", pt: "min de leitura" },
  "projects.title": { en: "Projects", pt: "Projetos" },
  "projects.featured": { en: "Featured", pt: "Destaque" },
  "experience.title": { en: "Experience", pt: "Experiencia" },
  "contact.title": { en: "Contact", pt: "Contato" },
  "contact.subtitle": {
    en: "Feel free to reach out. I'd love to hear from you.",
    pt: "Fique a vontade para entrar em contato.",
  },
  "contact.name": { en: "Name", pt: "Nome" },
  "contact.email": { en: "Email", pt: "Email" },
  "contact.message": { en: "Message", pt: "Mensagem" },
  "contact.send": { en: "Send Message", pt: "Enviar Mensagem" },
  "contact.success": {
    en: "Message sent! I'll get back to you soon.",
    pt: "Mensagem enviada! Retornarei em breve.",
  },
  "contact.or_email": {
    en: "Or email me directly at",
    pt: "Ou me envie um email diretamente em",
  },
  "footer.built_with": { en: "Built with Go & React", pt: "Feito com Go & React" },
};

const LanguageContext = createContext<LanguageContextType | null>(null);

function detectLanguage(): Lang {
  const stored = localStorage.getItem("lang");
  if (stored === "en" || stored === "pt") return stored;
  return "en";
}

export function LanguageProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Lang>(detectLanguage);

  const setLang = (l: Lang) => {
    setLangState(l);
    localStorage.setItem("lang", l);
  };

  useEffect(() => {
    document.documentElement.lang = lang;
  }, [lang]);

  const t = (key: string): string => {
    return translations[key]?.[lang] ?? key;
  };

  return (
    <LanguageContext.Provider value={{ lang, setLang, t }}>
      {children}
    </LanguageContext.Provider>
  );
}

export function useLanguage() {
  const ctx = useContext(LanguageContext);
  if (!ctx) throw new Error("useLanguage must be used within LanguageProvider");
  return ctx;
}
