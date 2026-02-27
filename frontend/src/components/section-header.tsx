import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

interface SectionHeaderProps {
  title: string;
  as?: "h2" | "h3";
  children?: ReactNode;
}

export function SectionHeader({ title, as = "h2", children }: SectionHeaderProps) {
  const Tag = as;

  return (
    <div
      className={cn(
        "flex items-center border-b border-border bg-surface/50 px-4 py-2",
        children && "justify-between"
      )}
    >
      <Tag
        className={cn(
          "font-semibold text-foreground",
          as === "h2" ? "text-lg" : "text-sm"
        )}
      >
        {title}
      </Tag>
      {children}
    </div>
  );
}
