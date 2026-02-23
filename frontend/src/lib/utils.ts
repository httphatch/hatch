import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";
import { Browser } from "@wailsio/runtime";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function safeOpenURL(url: string) {
  try {
    const parsed = new URL(url);
    if (!/^https?:$/i.test(parsed.protocol)) return;
    const host = parsed.hostname;
    if (/^(localhost|127\.|0\.0\.0\.0|::1|\[::1\]|10\.|192\.168\.|172\.(1[6-9]|2[0-9]|3[01])\.)/i.test(host)) return;
    Browser.OpenURL(parsed.toString());
  } catch {
    // Invalid URL, do nothing
  }
}
