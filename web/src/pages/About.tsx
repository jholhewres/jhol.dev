import { useEffect, useState } from "react";
import { useLanguage } from "../context/LanguageContext";
import { useSEO } from "../hooks/useSEO";
import BackNav from "../components/BackNav";
import { ContentSkeleton } from "../components/Skeleton";

export default function About() {
  const { lang, t } = useLanguage();
  const [html, setHtml] = useState("");
  const [loading, setLoading] = useState(true);
  useSEO({ title: "About", description: "Learn more about Jhol Hewres — AI Engineer.", url: "/about" });

  useEffect(() => {
    setLoading(true);
    fetch(`/api/about?lang=${lang}`)
      .then((r) => r.json())
      .then((data) => setHtml(data.html ?? ""))
      .catch(() => setHtml(""))
      .finally(() => setLoading(false));
  }, [lang]);

  return (
    <div className="py-8">
      <BackNav />
      <h1 className="text-2xl font-bold mb-6">{t("nav.about")}</h1>
      {loading ? (
        <ContentSkeleton />
      ) : (
        <div className="prose animate-fade-in" dangerouslySetInnerHTML={{ __html: html }} />
      )}
    </div>
  );
}
