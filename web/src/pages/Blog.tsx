import { useEffect, useState } from "react";
import { useLanguage } from "../context/LanguageContext";
import PostCard from "../components/PostCard";
import BackNav from "../components/BackNav";

interface PostSummary {
  slug: string;
  title: string;
  date: string;
  tags: string[];
  summary: string;
  reading_time: number;
}

export default function Blog() {
  const { lang, t } = useLanguage();
  const [posts, setPosts] = useState<PostSummary[]>([]);
  const [filterTag, setFilterTag] = useState<string | null>(null);

  useEffect(() => {
    fetch(`/api/posts?lang=${lang}`)
      .then((r) => r.json())
      .then((data) => setPosts(data ?? []))
      .catch(() => setPosts([]));
  }, [lang]);

  const allTags = [...new Set(posts.flatMap((p) => p.tags ?? []))].sort();
  const filtered = filterTag
    ? posts.filter((p) => p.tags?.includes(filterTag))
    : posts;

  return (
    <div className="py-8">
      <BackNav />
      <h1 className="text-2xl font-bold mb-6">{t("blog.title")}</h1>

      {allTags.length > 0 && (
        <div className="flex flex-wrap gap-2 mb-8">
          <button
            onClick={() => setFilterTag(null)}
            className={`text-xs px-2.5 py-1 rounded transition-colors ${
              !filterTag
                ? "bg-accent text-white"
                : "bg-gray-100 text-gray-600 hover:bg-gray-200"
            }`}
          >
            All
          </button>
          {allTags.map((tag) => (
            <button
              key={tag}
              onClick={() => setFilterTag(tag === filterTag ? null : tag)}
              className={`text-xs px-2.5 py-1 rounded transition-colors ${
                tag === filterTag
                  ? "bg-accent text-white"
                  : "bg-gray-100 text-gray-600 hover:bg-gray-200"
              }`}
            >
              {tag}
            </button>
          ))}
        </div>
      )}

      <div className="space-y-8">
        {filtered.map((post) => (
          <PostCard
            key={post.slug}
            slug={post.slug}
            title={post.title}
            date={post.date}
            summary={post.summary}
            tags={post.tags}
            readingTime={post.reading_time}
          />
        ))}
      </div>

      {filtered.length === 0 && (
        <p className="text-gray-500 text-center py-12">No posts yet.</p>
      )}
    </div>
  );
}
