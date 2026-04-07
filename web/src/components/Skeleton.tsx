export function PostSkeleton() {
  return (
    <div className="animate-pulse space-y-3">
      <div className="h-5 bg-[var(--color-border)] rounded w-3/4" />
      <div className="h-3 bg-[var(--color-border-subtle)] rounded w-1/3" />
      <div className="h-4 bg-[var(--color-border-subtle)] rounded w-full" />
      <div className="h-4 bg-[var(--color-border-subtle)] rounded w-5/6" />
    </div>
  );
}

export function PostListSkeleton({ count = 3 }: { count?: number }) {
  return (
    <div className="space-y-8">
      {Array.from({ length: count }).map((_, i) => (
        <PostSkeleton key={i} />
      ))}
    </div>
  );
}

export function ProjectSkeleton() {
  return (
    <div className="animate-pulse border border-[var(--color-border-subtle)] rounded-lg p-4 space-y-2">
      <div className="h-5 bg-[var(--color-border)] rounded w-1/2" />
      <div className="h-4 bg-[var(--color-border-subtle)] rounded w-full" />
      <div className="flex gap-2 mt-2">
        <div className="h-5 bg-[var(--color-border-subtle)] rounded w-12" />
        <div className="h-5 bg-[var(--color-border-subtle)] rounded w-16" />
        <div className="h-5 bg-[var(--color-border-subtle)] rounded w-10" />
      </div>
    </div>
  );
}

export function ProjectListSkeleton({ count = 3 }: { count?: number }) {
  return (
    <div className="grid gap-4">
      {Array.from({ length: count }).map((_, i) => (
        <ProjectSkeleton key={i} />
      ))}
    </div>
  );
}

export function ExperienceSkeleton() {
  return (
    <div className="animate-pulse relative pl-8 space-y-2">
      <div className="absolute left-0 top-2 w-[15px] h-[15px] rounded-full bg-[var(--color-border)]" />
      <div className="h-5 bg-[var(--color-border)] rounded w-2/3" />
      <div className="h-4 bg-[var(--color-border-subtle)] rounded w-1/3" />
      <div className="h-3 bg-[var(--color-border-subtle)] rounded w-1/4" />
      <div className="h-4 bg-[var(--color-border-subtle)] rounded w-full" />
    </div>
  );
}

export function ExperienceListSkeleton({ count = 3 }: { count?: number }) {
  return (
    <div className="relative">
      <div className="absolute left-[7px] top-2 bottom-2 w-px bg-[var(--color-border)]" />
      <div className="space-y-8">
        {Array.from({ length: count }).map((_, i) => (
          <ExperienceSkeleton key={i} />
        ))}
      </div>
    </div>
  );
}

export function ContentSkeleton() {
  return (
    <div className="animate-pulse space-y-4">
      <div className="h-6 bg-[var(--color-border)] rounded w-1/2" />
      <div className="h-4 bg-[var(--color-border-subtle)] rounded w-full" />
      <div className="h-4 bg-[var(--color-border-subtle)] rounded w-5/6" />
      <div className="h-4 bg-[var(--color-border-subtle)] rounded w-4/6" />
      <div className="h-4 bg-[var(--color-border-subtle)] rounded w-full" />
      <div className="h-4 bg-[var(--color-border-subtle)] rounded w-3/4" />
    </div>
  );
}

export function Spinner() {
  return (
    <div className="flex justify-center py-16">
      <div className="h-6 w-6 border-2 border-[var(--color-border)] border-t-accent rounded-full animate-spin" />
    </div>
  );
}
