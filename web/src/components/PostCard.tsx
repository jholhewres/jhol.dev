import { Link } from "react-router-dom";
import { useLanguage } from "../context/LanguageContext";

interface PostCardProps {
  slug: string;
  title: string;
  date: string;
  summary: string;
  tags: string[];
  readingTime: number;
}

export default function PostCard({
  slug,
  title,
  date,
  summary,
  tags,
  readingTime,
}: PostCardProps) {
  const { t } = useLanguage();

  return (
    <article className="group">
      <Link to={`/blog/${slug}`} className="block">
        <div className="flex items-center gap-2 text-sm text-gray-500 mb-1">
          <time dateTime={date}>
            {new Date(date + "T00:00:00").toLocaleDateString("en-US", {
              year: "numeric",
              month: "short",
              day: "numeric",
            })}
          </time>
          <span>&middot;</span>
          <span>
            {readingTime} {t("blog.reading_time")}
          </span>
        </div>
        <h3 className="text-lg font-semibold text-[#1a1a1a] group-hover:text-accent transition-colors mb-1">
          {title}
        </h3>
        <p className="text-gray-600 text-sm mb-2">{summary}</p>
        {tags && tags.length > 0 && (
          <div className="flex flex-wrap gap-1.5">
            {tags.map((tag) => (
              <span
                key={tag}
                className="text-xs bg-gray-100 text-gray-600 px-2 py-0.5 rounded"
              >
                {tag}
              </span>
            ))}
          </div>
        )}
      </Link>
    </article>
  );
}
