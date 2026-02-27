import { cn } from "@/lib/utils";

interface ErrorStateProps {
  message: string;
  className?: string;
}

export function ErrorState({ message, className }: ErrorStateProps) {
  return (
    <p className={cn("py-16 text-center text-destructive", className)}>
      {message}
    </p>
  );
}
