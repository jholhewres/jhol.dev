import { useEffect, useState, useCallback } from "react";

interface Stats {
  total: number;
  by_country: Record<string, number>;
  by_path: Record<string, number>;
  by_day: Record<string, number>;
  by_referer: Record<string, number>;
  window_hours: number;
}

function top10(obj: Record<string, number>): [string, number][] {
  return Object.entries(obj)
    .sort(([, a], [, b]) => b - a)
    .slice(0, 10);
}

function last7Days(byDay: Record<string, number>): [string, number][] {
  const days: [string, number][] = [];
  for (let i = 6; i >= 0; i--) {
    const d = new Date();
    d.setUTCDate(d.getUTCDate() - i);
    const key = d.toISOString().slice(0, 10);
    days.push([key, byDay[key] ?? 0]);
  }
  return days;
}

function Table({
  title,
  rows,
}: {
  title: string;
  rows: [string, number][];
}) {
  if (rows.length === 0) return null;
  return (
    <div className="mb-8">
      <h2 className="text-lg font-semibold text-[var(--color-fg)] mb-3">{title}</h2>
      <table className="w-full text-sm border-collapse">
        <tbody>
          {rows.map(([key, count]) => (
            <tr key={key} className="border-b border-[var(--color-border-subtle)]">
              <td className="py-1.5 pr-4 font-mono text-[var(--color-fg-muted)] truncate max-w-[300px]">
                {key}
              </td>
              <td className="py-1.5 text-right text-[var(--color-fg)] font-medium w-16">
                {count}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export default function AdminStats() {
  const [token, setToken] = useState<string>(
    () => sessionStorage.getItem("admin_token") ?? ""
  );
  const [input, setInput] = useState("");
  const [stats, setStats] = useState<Stats | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchStats = useCallback(
    async (t: string) => {
      setLoading(true);
      setError(null);
      try {
        const res = await fetch("/api/admin/stats", {
          headers: { "X-Admin-Token": t },
        });
        if (res.status === 401) {
          sessionStorage.removeItem("admin_token");
          setToken("");
          setStats(null);
          setError("Invalid token. Please try again.");
          return;
        }
        if (!res.ok) {
          setError(`Server error: ${res.status}`);
          return;
        }
        const data: Stats = await res.json();
        setStats(data);
      } catch (e) {
        setError("Network error. Check that the server is running.");
      } finally {
        setLoading(false);
      }
    },
    []
  );

  useEffect(() => {
    if (token) {
      fetchStats(token);
    }
  }, [token, fetchStats]);

  // Token prompt
  if (!token) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[60vh]">
        <div className="w-full max-w-sm">
          <h1 className="text-xl font-bold text-[var(--color-fg)] mb-6">Admin Stats</h1>
          {error && (
            <p className="text-red-600 text-sm mb-4">{error}</p>
          )}
          <label className="block text-sm text-[var(--color-fg-muted)] mb-1">
            Admin Token
          </label>
          <input
            type="password"
            className="w-full border border-[var(--color-border)] rounded px-3 py-2 text-sm mb-4 focus:outline-none focus:border-[var(--color-fg-muted)] bg-[var(--color-bg)] text-[var(--color-fg)]"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && input.trim()) {
                sessionStorage.setItem("admin_token", input.trim());
                setToken(input.trim());
              }
            }}
            autoFocus
            placeholder="Enter admin token"
          />
          <button
            className="w-full bg-[var(--color-fg)] text-[var(--color-bg)] text-sm py-2 rounded hover:opacity-90 transition-opacity"
            onClick={() => {
              if (input.trim()) {
                sessionStorage.setItem("admin_token", input.trim());
                setToken(input.trim());
              }
            }}
          >
            Submit
          </button>
        </div>
      </div>
    );
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-xl font-bold text-[var(--color-fg)]">Admin Stats</h1>
          {stats && (
            <p className="text-sm text-[var(--color-fg-subtle)] mt-0.5">
              Last {stats.window_hours}h — {stats.total} page views
            </p>
          )}
        </div>
        <div className="flex gap-2">
          <button
            className="text-sm px-3 py-1.5 border border-[var(--color-border)] rounded hover:bg-[var(--color-card)] transition-colors text-[var(--color-fg)]"
            onClick={() => fetchStats(token)}
            disabled={loading}
          >
            {loading ? "Loading…" : "Refresh"}
          </button>
          <button
            className="text-sm px-3 py-1.5 border border-[var(--color-border)] rounded hover:bg-[var(--color-card)] transition-colors text-[var(--color-fg-subtle)]"
            onClick={() => {
              sessionStorage.removeItem("admin_token");
              setToken("");
              setStats(null);
              setInput("");
            }}
          >
            Logout
          </button>
        </div>
      </div>

      {error && (
        <p className="text-red-600 text-sm mb-6">{error}</p>
      )}

      {loading && !stats && (
        <p className="text-[var(--color-fg-subtle)] text-sm">Loading…</p>
      )}

      {stats && (
        <>
          <Table title="Top Countries" rows={top10(stats.by_country)} />
          <Table title="Top Paths" rows={top10(stats.by_path)} />
          <Table title="Top Referers" rows={top10(stats.by_referer)} />
          <div className="mb-8">
            <h2 className="text-lg font-semibold text-[var(--color-fg)] mb-3">
              By Day (last 7)
            </h2>
            <table className="w-full text-sm border-collapse">
              <tbody>
                {last7Days(stats.by_day).map(([day, count]) => (
                  <tr key={day} className="border-b border-[var(--color-border-subtle)]">
                    <td className="py-1.5 pr-4 font-mono text-[var(--color-fg-muted)]">
                      {day}
                    </td>
                    <td className="py-1.5 text-right text-[var(--color-fg)] font-medium w-16">
                      {count}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </div>
  );
}
