import { useEffect, useState } from "react";
import { useLanguage } from "../context/LanguageContext";
import BackNav from "../components/BackNav";

export default function About() {
  const { lang, t } = useLanguage();
  const [html, setHtml] = useState("");

  useEffect(() => {
    fetch(`/api/about?lang=${lang}`)
      .then((r) => r.json())
      .then((data) => setHtml(data.html ?? ""))
      .catch(() => setHtml(""));
  }, [lang]);

  return (
    <div className="py-8">
      <BackNav />
      <h1 className="text-2xl font-bold mb-6">{t("nav.about")}</h1>
      <div className="prose" dangerouslySetInnerHTML={{ __html: html }} />
    </div>
  );
}
