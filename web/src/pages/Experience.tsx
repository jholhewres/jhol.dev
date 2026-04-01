import { useEffect, useState } from "react";
import { useLanguage } from "../context/LanguageContext";
import { useSEO } from "../hooks/useSEO";
import BackNav from "../components/BackNav";
import { ExperienceListSkeleton } from "../components/Skeleton";

interface ExperienceItem {
  role: string;
  company: string;
  period: string;
  description: string;
  tags: string[];
}

export default function Experience() {
  const { lang, t } = useLanguage();
  const [items, setItems] = useState<ExperienceItem[]>([]);
  const [loading, setLoading] = useState(true);
  useSEO({ title: "Experience", description: "Professional experience and career history of Jhol Hewres.", url: "/experience" });

  useEffect(() => {
    setLoading(true);
    fetch(`/api/experience?lang=${lang}`)
      .then((r) => r.json())
      .then((data) => setItems(data ?? []))
      .catch(() => setItems([]))
      .finally(() => setLoading(false));
  }, [lang]);

  return (
    <div className="py-8">
      <BackNav />
      <h1 className="text-2xl font-bold mb-8">{t("experience.title")}</h1>

      {loading ? (
        <ExperienceListSkeleton count={3} />
      ) : items.length > 0 ? (
        <div className="relative animate-fade-in">
          {/* Timeline line */}
          <div className="absolute left-[7px] top-2 bottom-2 w-px bg-gray-200" />

          <div className="space-y-8">
            {items.map((item, i) => (
              <div key={i} className="relative pl-8">
                {/* Timeline dot */}
                <div className="absolute left-0 top-2 w-[15px] h-[15px] rounded-full border-2 border-accent bg-white" />

                <div>
                  <h3 className="font-semibold text-[#1a1a1a]">{item.role}</h3>
                  <div className="text-sm text-gray-600 mt-0.5">
                    {item.company}
                  </div>
                  <div className="text-xs text-gray-400 mt-0.5">
                    {item.period}
                  </div>
                  <p className="text-sm text-gray-600 mt-2">
                    {item.description}
                  </p>
                  {item.tags && item.tags.length > 0 && (
                    <div className="flex flex-wrap gap-1.5 mt-2">
                      {item.tags.map((tag) => (
                        <span
                          key={tag}
                          className="text-xs bg-gray-100 text-gray-600 px-2 py-0.5 rounded"
                        >
                          {tag}
                        </span>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      ) : (
        <p className="text-gray-500 text-center py-12">No experience listed.</p>
      )}
    </div>
  );
}
