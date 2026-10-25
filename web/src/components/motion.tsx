"use client";
// Motion primitives for the app. LazyMotion keeps the initial bundle small
// (features load after first paint); MotionConfig honours the OS
// "reduce motion" setting for every animation below.
import {
  animate,
  AnimatePresence,
  LazyMotion,
  m,
  MotionConfig,
  useInView,
  useReducedMotion,
} from "motion/react";
import { useEffect, useRef, useState, type ReactNode } from "react";

const loadFeatures = () =>
  import("./motion-features").then((module) => module.default);

export const ease = [0.2, 0.8, 0.2, 1] as const;
export const spring = { type: "spring", stiffness: 520, damping: 34 } as const;

export function MotionProvider({ children }: { children: ReactNode }) {
  return (
    <LazyMotion features={loadFeatures} strict>
      <MotionConfig reducedMotion="user">{children}</MotionConfig>
    </LazyMotion>
  );
}

// Each route enters with a short rise; keyed by path so it replays on
// navigation but not on background refetches.
export function PageTransition({
  routeKey,
  children,
}: {
  routeKey: string;
  children: ReactNode;
}) {
  return (
    <m.div
      key={routeKey}
      className="page-transition"
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.24, ease }}
    >
      {children}
    </m.div>
  );
}

// Numbers count up when they first appear and glide to new values after
// live updates, so changes on the dashboard are noticed.
export function CountUp({ value }: { value: number }) {
  const ref = useRef<HTMLSpanElement>(null);
  const inView = useInView(ref, { once: true });
  const reduce = useReducedMotion();
  const shown = useRef(0);
  const [text, setText] = useState(() => format(reduce ? value : 0));
  useEffect(() => {
    if (!inView) return;
    if (reduce) {
      shown.current = value;
      setText(format(value));
      return;
    }
    const controls = animate(shown.current, value, {
      duration: Math.min(0.9, 0.35 + Math.abs(value - shown.current) * 0.02),
      ease,
      onUpdate: (latest) => {
        shown.current = latest;
        setText(format(Math.round(latest)));
      },
    });
    return () => controls.stop();
  }, [value, inView, reduce]);
  return (
    <span ref={ref} aria-label={format(value)}>
      <span aria-hidden>{text}</span>
    </span>
  );
}
function format(value: number) {
  return new Intl.NumberFormat("ru-RU").format(value);
}

// Horizontal bar that grows from zero with a spring (or snaps to its width
// when the OS asks for reduced motion).
export function GrowBar({
  share,
  className,
  empty,
}: {
  share: number;
  className: string;
  empty?: boolean;
}) {
  const reduce = useReducedMotion();
  return (
    <m.span
      className={className}
      data-empty={empty ? "" : undefined}
      initial={reduce ? false : { width: 0 }}
      animate={{ width: `${share}%` }}
      transition={
        reduce
          ? { duration: 0 }
          : { type: "spring", stiffness: 140, damping: 22, mass: 0.8 }
      }
    />
  );
}

// Conditionally rendered blocks (callouts, extra fields, status lines) grow
// and fade in instead of popping, and shrink away before unmounting.
// `initial={false}`: content present on first render does not animate.
// Overflow is clipped only while the height animates, so focus rings and
// offset shadows inside are not cut once the block has settled.
export function AnimatedCollapse({
  show,
  children,
  className,
}: {
  show: boolean;
  children: ReactNode;
  className?: string;
}) {
  const reduce = useReducedMotion();
  return (
    <AnimatePresence initial={false}>
      {show ? (
        <m.div
          className={className}
          style={{ overflow: "hidden" }}
          initial={{ height: 0, opacity: 0 }}
          animate={{
            height: "auto",
            opacity: 1,
            transitionEnd: { overflow: "visible" },
          }}
          exit={{ height: 0, opacity: 0, overflow: "hidden" }}
          transition={reduce ? { duration: 0 } : { duration: 0.22, ease }}
        >
          {children}
        </m.div>
      ) : null}
    </AnimatePresence>
  );
}

// The yellow "you are here" block that slides between sidebar items.
export function NavIndicator() {
  return (
    <m.span
      className="nav-indicator"
      layoutId="nav-indicator"
      transition={spring}
      aria-hidden
    />
  );
}

export { m };
