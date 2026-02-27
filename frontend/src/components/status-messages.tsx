import { cn } from "@/lib/utils";

interface LoadingStateProps {
  message?: string;
  className?: string;
}

export function LoadingState({ message = "Loading...", className }: LoadingStateProps) {
  return (
    <p className={cn("py-16 text-center text-muted-foreground", className)}>
      {message}
    </p>
  );
}

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
