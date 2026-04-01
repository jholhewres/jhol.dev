import { useEffect } from "react";

interface SEOProps {
  title?: string;
  description?: string;
  url?: string;
  type?: string;
  image?: string;
}

const SITE_NAME = "Jhol Hewres";
const DEFAULT_DESC =
  "AI Engineer building production-ready AI systems. Multi-agent architecture, RAG & LLMOps.";
const BASE_URL = "https://jhol.dev";
const DEFAULT_IMAGE = `${BASE_URL}/avatar.jpg`;
const BLOG_IMAGE = `${BASE_URL}/og-blog.png`;

function setMeta(property: string, content: string, isOG = false) {
  const attr = isOG ? "property" : "name";
  let el = document.querySelector(`meta[${attr}="${property}"]`);
  if (!el) {
    el = document.createElement("meta");
    el.setAttribute(attr, property);
    document.head.appendChild(el);
  }
  el.setAttribute("content", content);
}

export function useSEO({ title, description, url, type, image }: SEOProps) {
  useEffect(() => {
    const fullTitle = title ? `${title} — ${SITE_NAME}` : `${SITE_NAME} — AI & Software Engineer`;
    const desc = description || DEFAULT_DESC;
    const pageUrl = url ? `${BASE_URL}${url}` : BASE_URL;
    const ogType = type || "website";
    const ogImage = image || (type === "article" ? BLOG_IMAGE : DEFAULT_IMAGE);

    document.title = fullTitle;

    setMeta("description", desc);

    let canonical = document.querySelector('link[rel="canonical"]') as HTMLLinkElement | null;
    if (!canonical) {
      canonical = document.createElement("link");
      canonical.rel = "canonical";
      document.head.appendChild(canonical);
    }
    canonical.href = pageUrl;

    setMeta("og:title", fullTitle, true);
    setMeta("og:description", desc, true);
    setMeta("og:url", pageUrl, true);
    setMeta("og:type", ogType, true);
    setMeta("og:image", ogImage, true);

    setMeta("twitter:title", fullTitle);
    setMeta("twitter:description", desc);
    setMeta("twitter:image", ogImage);
  }, [title, description, url, type, image]);
}
