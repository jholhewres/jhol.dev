import { useEffect, useState } from "react";

interface Props {
  slug: string;
}

export default function LikeButton({ slug }: Props) {
  const [count, setCount] = useState(0);
  const [liked, setLiked] = useState(false);
  const [animating, setAnimating] = useState(false);

  useEffect(() => {
    fetch(`/api/posts/${slug}/likes`)
      .then((r) => r.json())
      .then((data) => setCount(data.likes ?? 0))
      .catch(() => {});

    const stored = localStorage.getItem(`liked:${slug}`);
    if (stored === "1") setLiked(true);
  }, [slug]);

  const handleLike = async () => {
    if (liked) return;

    setAnimating(true);
    setTimeout(() => setAnimating(false), 300);

    try {
      const res = await fetch(`/api/posts/${slug}/like`, { method: "POST" });
      const data = await res.json();
      setCount(data.likes);
      setLiked(true);
      localStorage.setItem(`liked:${slug}`, "1");
    } catch {
      // Silently fail
    }
  };

  return (
    <button
      onClick={handleLike}
      className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-sm transition-all border ${
        liked
          ? "bg-red-50 border-red-200 text-red-500"
          : "bg-white border-gray-200 text-gray-500 hover:border-red-200 hover:text-red-400"
      }`}
      aria-label={liked ? "Liked" : "Like this post"}
    >
      <svg
        className={`w-4 h-4 ${animating ? "animate-like-pop" : ""}`}
        fill={liked ? "currentColor" : "none"}
        stroke="currentColor"
        viewBox="0 0 24 24"
        strokeWidth={2}
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          d="M21 8.25c0-2.485-2.099-4.5-4.688-4.5-1.935 0-3.597 1.126-4.312 2.733-.715-1.607-2.377-2.733-4.313-2.733C5.1 3.75 3 5.765 3 8.25c0 7.22 9 12 9 12s9-4.78 9-12z"
        />
      </svg>
      <span>{count > 0 ? count : ""}</span>
    </button>
  );
}
