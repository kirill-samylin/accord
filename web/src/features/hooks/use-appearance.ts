"use client";
import { useEffect, useState } from "react";
export function useAppearance() {
  const [dark, setDark] = useState(false);
  useEffect(() => {
    let selected: string | null = null;
    try {
      selected = localStorage.getItem("accord.appearance.v1");
    } catch {
      /* Optional preference. */
    }
    const value =
      selected === "dark" ||
      (!selected && matchMedia("(prefers-color-scheme: dark)").matches);
    setDark(value);
    document.documentElement.dataset.theme = value ? "dark" : "light";
  }, []);
  function toggle() {
    const value = !dark;
    setDark(value);
    document.documentElement.dataset.theme = value ? "dark" : "light";
    try {
      localStorage.setItem("accord.appearance.v1", value ? "dark" : "light");
    } catch {
      /* Optional preference. */
    }
  }
  return { dark, toggle };
}
