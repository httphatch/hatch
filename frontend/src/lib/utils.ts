import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";
import { Browser } from "@wailsio/runtime";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function safeOpenURL(url: string) {
  if (/^https?:\/\//i.test(url)) {
    Browser.OpenURL(url);
  }
}
