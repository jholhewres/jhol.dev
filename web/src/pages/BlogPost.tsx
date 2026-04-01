import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { useLanguage } from "../context/LanguageContext";
import { useSEO } from "../hooks/useSEO";
import LikeButton from "../components/LikeButton";

interface Post {
  slug: string;
  title: string;
  date: string;
  tags: string[];
  summary: string;
  reading_time: number;
  content: string;
}

export default function BlogPost() {
  const { slug } = useParams<{ slug: string }>();
  const { lang, t } = useLanguage();
  const [post, setPost] = useState<Post | null>(null);
  const [notFound, setNotFound] = useState(false);
  const [readingMode, setReadingMode] = useState(false);
  useSEO(post ? {
    title: post.title,
    description: post.summary,
    url: `/blog/${post.slug}`,
    type: "article",
  } : {});

  useEffect(() => {
    fetch(`/api/posts/${slug}?lang=${lang}`)
      .then((r) => {
        if (!r.ok) throw new Error("not found");
        return r.json();
      })
      .then((data) => {
        setPost(data);
        setNotFound(false);
      })
      .catch(() => setNotFound(true));
  }, [slug, lang]);

  if (notFound) {
    return (
      <div className="py-16 text-center">
        <h1 className="text-2xl font-bold mb-4">Post not found</h1>
        <Link to="/blog" className="text-accent hover:text-accent-hover">
          &larr; {t("home.view_all")}
        </Link>
      </div>
    );
  }

  if (!post) {
    return (
      <div className="py-8 animate-pulse space-y-4">
        <div className="h-4 bg-gray-200 rounded w-16" />
        <div className="h-8 bg-gray-200 rounded w-3/4 mt-4" />
        <div className="h-3 bg-gray-100 rounded w-1/3" />
        <div className="space-y-3 mt-8">
          <div className="h-4 bg-gray-100 rounded w-full" />
          <div className="h-4 bg-gray-100 rounded w-5/6" />
          <div className="h-4 bg-gray-100 rounded w-4/6" />
          <div className="h-4 bg-gray-100 rounded w-full" />
          <div className="h-4 bg-gray-100 rounded w-3/4" />
        </div>
      </div>
    );
  }

  const jsonLd = {
    "@context": "https://schema.org",
    "@type": "BlogPosting",
    headline: post.title,
    description: post.summary,
    datePublished: post.date,
    author: { "@type": "Person", name: "Jhol Hewres" },
    url: `https://jhol.dev/blog/${post.slug}`,
    keywords: post.tags?.join(", "),
  };

  return (
    <article className={`py-8 animate-fade-in transition-all duration-300 ${readingMode ? "reading-mode" : ""}`}>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
      />

      <div className={`flex items-center justify-between ${readingMode ? "mb-6" : ""}`}>
        <Link
          to="/blog"
          className={`text-sm text-accent hover:text-accent-hover transition-colors ${readingMode ? "opacity-0 pointer-events-none" : ""}`}
        >
          &larr; {t("nav.blog")}
        </Link>

        <button
          onClick={() => setReadingMode(!readingMode)}
          className="inline-flex items-center gap-1.5 text-xs text-gray-400 hover:text-gray-600 transition-colors px-2 py-1 rounded border border-transparent hover:border-gray-200"
          aria-label={readingMode ? "Exit reading mode" : "Enter reading mode"}
          title={readingMode ? "Exit reading mode" : "Reading mode"}
        >
          <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={1.5}>
            {readingMode ? (
              <path strokeLinecap="round" strokeLinejoin="round" d="M9 9V4.5M9 9H4.5M9 9L3.75 3.75M9 15v4.5M9 15H4.5M9 15l-5.25 5.25M15 9h4.5M15 9V4.5M15 9l5.25-5.25M15 15h4.5M15 15v4.5m0-4.5l5.25 5.25" />
            ) : (
              <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 3.75v4.5m0-4.5h4.5m-4.5 0L9 9M3.75 20.25v-4.5m0 4.5h4.5m-4.5 0L9 15M20.25 3.75h-4.5m4.5 0v4.5m0-4.5L15 9m5.25 11.25h-4.5m4.5 0v-4.5m0 4.5L15 15" />
            )}
          </svg>
          {readingMode ? "Exit" : "Read"}
        </button>
      </div>

      <header className={`mt-4 mb-8 transition-all duration-300 ${readingMode ? "text-center mb-12" : ""}`}>
        <h1 className={`font-bold mb-2 transition-all duration-300 ${readingMode ? "text-3xl" : "text-2xl"}`}>{post.title}</h1>
        <div className={`flex items-center gap-2 text-sm text-gray-500 ${readingMode ? "justify-center" : ""}`}>
          <time dateTime={post.date}>
            {new Date(post.date + "T00:00:00").toLocaleDateString("en-US", {
              year: "numeric",
              month: "long",
              day: "numeric",
            })}
          </time>
          <span>&middot;</span>
          <span>
            {post.reading_time} {t("blog.reading_time")}
          </span>
        </div>
        {!readingMode && post.tags && post.tags.length > 0 && (
          <div className="flex flex-wrap gap-1.5 mt-3">
            {post.tags.map((tag) => (
              <span
                key={tag}
                className="text-xs bg-gray-100 text-gray-600 px-2 py-0.5 rounded"
              >
                {tag}
              </span>
            ))}
          </div>
        )}
      </header>

      <div
        className={`prose transition-all duration-300 ${readingMode ? "reading-mode-prose" : ""}`}
        dangerouslySetInnerHTML={{ __html: post.content }}
      />

      <div className={`mt-10 pt-6 border-t border-gray-100 transition-opacity duration-300 ${readingMode ? "opacity-0 pointer-events-none h-0 mt-0 pt-0 border-0 overflow-hidden" : ""}`}>
        <LikeButton slug={post.slug} />
      </div>
    </article>
  );
}
