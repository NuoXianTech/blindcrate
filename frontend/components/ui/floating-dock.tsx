"use client";

import { Fragment, useRef, useState } from "react";
import Link from "next/link";
import { PanelBottomOpen } from "lucide-react";
import {
  AnimatePresence,
  motion,
  useMotionValue,
  useSpring,
  useTransform,
  type MotionValue,
} from "motion/react";
import { cn } from "cn";

export interface DockItem {
  title: string;
  icon: React.ReactNode;
  href?: string;
  onClick?: () => void;
  active?: boolean;
}

/** 用 null 分隔不同分组 */
export type DockEntry = DockItem | null;

export function FloatingDock({
  items,
  desktopClassName,
  mobileClassName,
}: {
  items: DockEntry[];
  desktopClassName?: string;
  mobileClassName?: string;
}) {
  return (
    <>
      <FloatingDockDesktop items={items} className={desktopClassName} />
      <FloatingDockMobile items={items} className={mobileClassName} />
    </>
  );
}

function ItemAction({
  item,
  className,
  children,
}: {
  item: DockItem;
  className?: string;
  children: React.ReactNode;
}) {
  if (item.href) {
    return (
      <Link href={item.href} aria-label={item.title} className={className} onClick={item.onClick}>
        {children}
      </Link>
    );
  }
  return (
    <button type="button" aria-label={item.title} className={className} onClick={item.onClick}>
      {children}
    </button>
  );
}

function FloatingDockMobile({ items, className }: { items: DockEntry[]; className?: string }) {
  const [open, setOpen] = useState(false);
  const flat = items.filter((item): item is DockItem => item !== null);

  return (
    <div className={cn("relative block md:hidden", className)}>
      <AnimatePresence>
        {open && (
          <motion.div layoutId="nav" className="absolute inset-x-0 bottom-full mb-2 flex flex-col gap-2">
            {flat.map((item, idx) => (
              <motion.div
                key={item.title}
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: 10, transition: { delay: idx * 0.05 } }}
                transition={{ delay: (flat.length - 1 - idx) * 0.05 }}
              >
                <ItemAction
                  item={{ ...item, onClick: () => (item.onClick?.(), setOpen(false)) }}
                  className={cn(
                    "flex size-10 items-center justify-center rounded-full border bg-popover text-muted-foreground",
                    item.active && "text-primary",
                  )}
                >
                  <span className="size-4">{item.icon}</span>
                </ItemAction>
              </motion.div>
            ))}
          </motion.div>
        )}
      </AnimatePresence>
      <button
        type="button"
        aria-label="展开导航"
        onClick={() => setOpen(!open)}
        className="flex size-10 items-center justify-center rounded-full border bg-card text-muted-foreground backdrop-blur-md"
      >
        <PanelBottomOpen className="size-5" />
      </button>
    </div>
  );
}

function FloatingDockDesktop({ items, className }: { items: DockEntry[]; className?: string }) {
  const mouseX = useMotionValue(Infinity);

  return (
    <motion.div
      onMouseMove={(e) => mouseX.set(e.pageX)}
      onMouseLeave={() => mouseX.set(Infinity)}
      className={cn(
        "mx-auto hidden h-16 items-end gap-3 rounded-2xl border bg-card px-4 pb-3 shadow-lg backdrop-blur-md md:flex",
        className,
      )}
    >
      {items.map((item, idx) =>
        item === null ? (
          <div key={`divider-${idx}`} className="mb-2 h-6 w-px self-end bg-border" />
        ) : (
          <Fragment key={item.title}>
            <IconContainer mouseX={mouseX} item={item} />
          </Fragment>
        ),
      )}
    </motion.div>
  );
}

const SPRING = { mass: 0.1, stiffness: 150, damping: 12 };

function IconContainer({ mouseX, item }: { mouseX: MotionValue<number>; item: DockItem }) {
  const ref = useRef<HTMLDivElement>(null);

  const distance = useTransform(mouseX, (val) => {
    const bounds = ref.current?.getBoundingClientRect() ?? { x: 0, width: 0 };
    return val - bounds.x - bounds.width / 2;
  });

  const size = useSpring(useTransform(distance, [-150, 0, 150], [40, 64, 40]), SPRING);
  const iconSize = useSpring(useTransform(distance, [-150, 0, 150], [20, 32, 20]), SPRING);

  const [hovered, setHovered] = useState(false);

  return (
    <ItemAction item={item}>
      <motion.div
        ref={ref}
        style={{ width: size, height: size }}
        onMouseEnter={() => setHovered(true)}
        onMouseLeave={() => setHovered(false)}
        className={cn(
          "relative flex aspect-square items-center justify-center rounded-full bg-secondary text-muted-foreground transition-colors hover:text-foreground",
          item.active && "bg-primary/15 text-primary hover:text-primary",
        )}
      >
        <AnimatePresence>
          {hovered && (
            <motion.div
              initial={{ opacity: 0, y: 10, x: "-50%" }}
              animate={{ opacity: 1, y: 0, x: "-50%" }}
              exit={{ opacity: 0, y: 2, x: "-50%" }}
              className="absolute -top-8 left-1/2 w-fit rounded-md border bg-popover px-2 py-0.5 text-xs whitespace-pre text-popover-foreground"
            >
              {item.title}
            </motion.div>
          )}
        </AnimatePresence>
        <motion.div style={{ width: iconSize, height: iconSize }} className="flex items-center justify-center [&>svg]:size-full">
          {item.icon}
        </motion.div>
      </motion.div>
    </ItemAction>
  );
}
