import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { useLanguage } from "../context/LanguageContext";

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
    return <div className="py-16 text-center text-gray-400">Loading...</div>;
  }

  return (
    <article className="py-8">
      <Link
        to="/blog"
        className="text-sm text-accent hover:text-accent-hover transition-colors"
      >
        &larr; {t("nav.blog")}
      </Link>

      <header className="mt-4 mb-8">
        <h1 className="text-2xl font-bold mb-2">{post.title}</h1>
        <div className="flex items-center gap-2 text-sm text-gray-500">
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
        {post.tags && post.tags.length > 0 && (
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
        className="prose"
        dangerouslySetInnerHTML={{ __html: post.content }}
      />
    </article>
  );
}
