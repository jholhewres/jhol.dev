import { useEffect, useState } from "react";
import { useLanguage } from "../context/LanguageContext";
import { useSEO } from "../hooks/useSEO";
import BackNav from "../components/BackNav";
import { ProjectListSkeleton } from "../components/Skeleton";

interface Project {
  name: string;
  description: string;
  tags: string[];
  url: string;
  featured: boolean;
}

export default function Projects() {
  const { lang, t } = useLanguage();
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  useSEO({ title: "Projects", description: "Open source projects and tools built by Jhol Hewres.", url: "/projects" });

  useEffect(() => {
    setLoading(true);
    fetch(`/api/projects?lang=${lang}`)
      .then((r) => r.json())
      .then((data) => setProjects(data ?? []))
      .catch(() => setProjects([]))
      .finally(() => setLoading(false));
  }, [lang]);

  const featured = projects.filter((p) => p.featured);
  const others = projects.filter((p) => !p.featured);

  return (
    <div className="py-8">
      <BackNav />
      <h1 className="text-2xl font-bold mb-6">{t("projects.title")}</h1>

      {loading ? (
        <ProjectListSkeleton count={4} />
      ) : (
        <div className="animate-fade-in">
          {featured.length > 0 && (
            <div className="mb-10">
              <h2 className="text-sm font-medium text-gray-500 uppercase tracking-wide mb-4">
                {t("projects.featured")}
              </h2>
              <div className="grid gap-4">
                {featured.map((p) => (
                  <ProjectCard key={p.name} project={p} />
                ))}
              </div>
            </div>
          )}

          {others.length > 0 && (
            <div className="grid gap-4">
              {others.map((p) => (
                <ProjectCard key={p.name} project={p} />
              ))}
            </div>
          )}

          {projects.length === 0 && (
            <p className="text-gray-500 text-center py-12">No projects yet.</p>
          )}
        </div>
      )}
    </div>
  );
}

function ProjectCard({ project }: { project: Project }) {
  return (
    <a
      href={project.url}
      target="_blank"
      rel="noopener noreferrer"
      className="block border border-gray-100 rounded-lg p-4 hover:border-gray-300 transition-colors group"
    >
      <div className="flex items-start justify-between gap-3">
        <div>
          <h3 className="font-semibold text-[#1a1a1a] group-hover:text-accent transition-colors">
            {project.name}
          </h3>
          <p className="text-sm text-gray-600 mt-1">{project.description}</p>
          {project.tags && project.tags.length > 0 && (
            <div className="flex flex-wrap gap-1.5 mt-2">
              {project.tags.map((tag) => (
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
        <svg
          className="w-4 h-4 text-gray-400 group-hover:text-accent transition-colors shrink-0 mt-1"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
          />
        </svg>
      </div>
    </a>
  );
}
