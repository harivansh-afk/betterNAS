"use client";

import { useState } from "react";
import Link from "next/link";

/* ------------------------------------------------------------------ */
/*  README content (rendered as simple markdown-ish HTML)              */
/* ------------------------------------------------------------------ */

const README_LINES = [
  { tag: "h1", text: "betterNAS" },
  {
    tag: "p",
    text: "Mount VMs and remote filesystems on your Mac as native Finder volumes. No special client, no syncing - just your files, where you expect them.",
  },
  {
    tag: "p",
    text: "Soon: a unified layer across your phone, computer, and AI agents. A safe, modular backup of your filesystem that you can use natively - and a way to deploy agents on your own infrastructure without giving up control.",
  },
] as const;

/* ------------------------------------------------------------------ */
/*  Icons                                                             */
/* ------------------------------------------------------------------ */

function GithubIcon({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 16 16" fill="currentColor" className={className}>
      <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z" />
    </svg>
  );
}

function ClockIcon() {
  return (
    <svg className="size-[18px] text-[#65a2f8]" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
      <circle cx="12" cy="12" r="10" />
      <path d="M12 6v6l4 2" />
    </svg>
  );
}

function SharedIcon() {
  return (
    <svg className="size-[18px] text-[#65a2f8]" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2" />
      <circle cx="9" cy="7" r="4" />
      <path d="M23 21v-2a4 4 0 00-3-3.87M16 3.13a4 4 0 010 7.75" />
    </svg>
  );
}

function LibraryIcon() {
  return (
    <svg className="size-[18px] text-[#65a2f8]" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M3 21h18M3 7v14M21 7v14M6 7V3h12v4M9 21V11M15 21V11" />
    </svg>
  );
}

function AppIcon() {
  return (
    <svg className="size-[18px] text-[#65a2f8]" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5" />
    </svg>
  );
}

function DesktopIcon() {
  return (
    <svg className="size-[18px] text-[#65a2f8]" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
      <rect x="2" y="3" width="20" height="14" rx="2" />
      <path d="M8 21h8M12 17v4" />
    </svg>
  );
}

function DownloadIcon() {
  return (
    <svg className="size-[18px] text-[#65a2f8]" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4M7 10l5 5 5-5M12 15V3" />
    </svg>
  );
}

function DocumentsIcon() {
  return (
    <svg className="size-[18px] text-[#65a2f8]" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z" />
      <polyline points="14,2 14,8 20,8" />
    </svg>
  );
}

function FolderIcon({ className }: { className?: string }) {
  return (
    <svg className={className ?? "size-[18px] text-[#65a2f8]"} viewBox="0 0 24 24" fill="currentColor">
      <path d="M2 6a2 2 0 012-2h5l2 2h9a2 2 0 012 2v10a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
    </svg>
  );
}

function CloudIcon() {
  return (
    <svg className="size-[18px]" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M18 10h-1.26A8 8 0 109 20h9a5 5 0 000-10z" />
    </svg>
  );
}

function HomeIcon() {
  return (
    <svg className="size-[18px] text-[#65a2f8]" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-4 0a1 1 0 01-1-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 01-1 1" />
    </svg>
  );
}

function NetworkIcon() {
  return (
    <svg className="size-[18px] text-[#65a2f8]" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
      <rect x="2" y="3" width="20" height="14" rx="2" />
      <path d="M8 21h8M12 17v4" />
    </svg>
  );
}

function AirdropIcon() {
  return (
    <svg className="size-[18px] text-[#65a2f8]" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
      <circle cx="12" cy="12" r="3" />
      <path d="M16.24 7.76a6 6 0 010 8.49M7.76 16.24a6 6 0 010-8.49M19.07 4.93a10 10 0 010 14.14M4.93 19.07a10 10 0 010-14.14" />
    </svg>
  );
}

/* ------------------------------------------------------------------ */
/*  README modal (Quick Look style)                                   */
/* ------------------------------------------------------------------ */

function ReadmeModal({ onClose }: { onClose: () => void }) {
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="relative mx-4 flex max-h-[80vh] w-full max-w-2xl flex-col overflow-hidden rounded-xl border border-border bg-card shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        {/* titlebar */}
        <div className="flex shrink-0 items-center justify-between border-b border-border px-4 py-3">
          <div className="flex items-center gap-2">
            <button
              onClick={onClose}
              className="flex size-3 items-center justify-center rounded-full bg-[#ff5f57] transition-opacity hover:opacity-80"
              aria-label="Close"
            />
            <span className="size-3 rounded-full bg-[#febc2e]" />
            <span className="size-3 rounded-full bg-[#28c840]" />
          </div>
          <span className="text-xs font-medium text-muted-foreground">
            README.md
          </span>
          <span className="w-[52px]" />
        </div>

        {/* body */}
        <div className="overflow-y-auto p-6">
          <div className="prose-sm max-w-none space-y-4 text-foreground">
            {README_LINES.map((block, i) => {
              if (block.tag === "h1")
                return (
                  <h1
                    key={i}
                    className="text-2xl font-bold tracking-tight text-foreground"
                  >
                    {block.text}
                  </h1>
                );
              return (
                <p key={i} className="text-sm leading-relaxed text-muted-foreground">
                  {block.text}
                </p>
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Finder sidebar item                                               */
/* ------------------------------------------------------------------ */

function SidebarItem({
  icon,
  label,
  active,
  accent,
  onClick,
}: {
  icon: React.ReactNode;
  label: string;
  active?: boolean;
  accent?: string;
  onClick?: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className={`flex w-full items-center gap-2.5 rounded-md px-2 py-[5px] text-left text-[13px] transition-colors ${
        active
          ? "bg-primary/15 text-foreground"
          : "text-muted-foreground hover:bg-muted/50"
      }`}
    >
      <span className={accent ?? ""}>{icon}</span>
      <span className="truncate">{label}</span>
    </button>
  );
}

/* ------------------------------------------------------------------ */
/*  Finder file grid item (folder)                                    */
/* ------------------------------------------------------------------ */

function GridFolder({
  name,
  itemCount,
  onClick,
}: {
  name: string;
  itemCount?: number;
  onClick?: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className="group flex flex-col items-center gap-1 rounded-lg p-3 transition-colors hover:bg-muted/50"
    >
      <svg className="size-16 text-[#3b9dff] drop-shadow-sm" viewBox="0 0 64 56" fill="currentColor">
        <path d="M2 8a6 6 0 016-2h14l4 4h30a6 6 0 016 6v32a6 6 0 01-6 6H8a6 6 0 01-6-6V8z" opacity="0.85" />
        <path d="M2 16h60v32a6 6 0 01-6 6H8a6 6 0 01-6-6V16z" opacity="0.95" />
      </svg>
      <span className="max-w-[100px] truncate text-xs text-foreground">
        {name}
      </span>
      {itemCount !== undefined && (
        <span className="text-[10px] text-muted-foreground">
          {itemCount} {itemCount === 1 ? "item" : "items"}
        </span>
      )}
    </button>
  );
}

/* ------------------------------------------------------------------ */
/*  Finder file grid item (file)                                      */
/* ------------------------------------------------------------------ */

function GridFile({
  name,
  meta,
  onClick,
}: {
  name: string;
  meta?: string;
  onClick?: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className="group flex flex-col items-center gap-1 rounded-lg p-3 transition-colors hover:bg-muted/50"
    >
      <div className="relative flex size-16 items-center justify-center">
        <svg className="size-14 text-muted-foreground/30" viewBox="0 0 48 56" fill="currentColor">
          <path d="M4 0h28l12 12v40a4 4 0 01-4 4H4a4 4 0 01-4-4V4a4 4 0 014-4z" />
          <path d="M32 0l12 12H36a4 4 0 01-4-4V0z" opacity="0.5" />
        </svg>
        <span className="absolute bottom-2 text-[9px] font-semibold uppercase tracking-wide text-foreground/60">
          MD
        </span>
      </div>
      <span className="max-w-[100px] truncate text-xs text-foreground">
        {name}
      </span>
      {meta && (
        <span className="text-[10px] text-muted-foreground">{meta}</span>
      )}
    </button>
  );
}

/* ------------------------------------------------------------------ */
/*  Main page                                                         */
/* ------------------------------------------------------------------ */

export default function LandingPage() {
  const [readmeOpen, setReadmeOpen] = useState(false);
  const [selectedSidebar, setSelectedSidebar] = useState("DAV");

  return (
    <div className="flex min-h-screen flex-col bg-background text-foreground">
      {/* ---- header ---- */}
      <header className="flex shrink-0 items-center justify-end px-5 py-3.5">
        <div className="flex items-center gap-2">
          <Link
            href="/docs"
            className="rounded-xl border border-border bg-muted/30 px-4 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          >
            Docs
          </Link>
          <Link
            href="/login"
            className="rounded-xl border border-border bg-muted/30 px-4 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          >
            Sign in
          </Link>
          <a
            href="https://github.com/harivansh-afk/betterNAS"
            target="_blank"
            rel="noopener noreferrer"
            className="flex size-8 items-center justify-center rounded-xl border border-border bg-muted/30 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            aria-label="GitHub"
          >
            <GithubIcon className="size-4" />
          </a>
        </div>
      </header>

      {/* ---- finder ---- */}
      <main className="flex flex-1 items-center justify-center p-4 sm:p-8">
        <div className="w-full max-w-4xl overflow-hidden rounded-xl border border-border bg-card shadow-xl">
          {/* titlebar */}
          <div className="flex items-center border-b border-border bg-muted/30 px-4 py-2.5">
            <div className="flex items-center gap-2">
              <span className="size-3 rounded-full bg-[#ff5f57]" />
              <span className="size-3 rounded-full bg-[#febc2e]" />
              <span className="size-3 rounded-full bg-[#28c840]" />
            </div>

            <div className="mx-auto flex items-center gap-2">
              <span className="text-sm font-medium text-foreground">DAV</span>
            </div>

            {/* forward/back placeholders */}
            <div className="flex items-center gap-1 text-muted-foreground/40">
              <svg className="size-4" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
                <path d="M10 3L5 8l5 5" />
              </svg>
              <svg className="size-4" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
                <path d="M6 3l5 5-5 5" />
              </svg>
            </div>
          </div>

          {/* content area */}
          <div className="flex min-h-[480px]">
            {/* ---- sidebar ---- */}
            <div className="hidden w-[180px] shrink-0 flex-col gap-0.5 border-r border-border bg-muted/20 p-3 sm:flex">
              {/* Favorites */}
              <p className="mb-1 mt-1 px-2 text-[10px] font-semibold uppercase tracking-widest text-muted-foreground/50">
                Favorites
              </p>
              <SidebarItem icon={<ClockIcon />} label="Recents" />
              <SidebarItem icon={<SharedIcon />} label="Shared" />
              <SidebarItem icon={<LibraryIcon />} label="Library" />
              <SidebarItem icon={<AppIcon />} label="Applications" />
              <SidebarItem icon={<DesktopIcon />} label="Desktop" />
              <SidebarItem icon={<DownloadIcon />} label="Downloads" />
              <SidebarItem icon={<DocumentsIcon />} label="Documents" />
              <SidebarItem icon={<FolderIcon className="size-[18px] text-[#65a2f8]" />} label="GitHub" />

              {/* Locations */}
              <p className="mb-1 mt-4 px-2 text-[10px] font-semibold uppercase tracking-widest text-muted-foreground/50">
                Locations
              </p>
              <SidebarItem icon={<HomeIcon />} label="rathi" />
              <SidebarItem icon={<NetworkIcon />} label="hari-macbook-pro" />
              <SidebarItem
                icon={<CloudIcon />}
                label="DAV"
                active={selectedSidebar === "DAV"}
                accent="text-[#65a2f8]"
                onClick={() => setSelectedSidebar("DAV")}
              />
              <SidebarItem icon={<AirdropIcon />} label="AirDrop" />
            </div>

            {/* ---- file grid ---- */}
            <div className="flex flex-1 flex-col">
              {/* toolbar */}
              <div className="flex items-center justify-between border-b border-border px-4 py-2">
                <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                  <CloudIcon />
                  <span className="font-medium text-foreground">DAV</span>
                  <span className="text-muted-foreground/50">/</span>
                  <span>exports</span>
                </div>
                <div className="flex items-center gap-2 text-muted-foreground/50">
                  <svg className="size-4" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
                    <rect x="1" y="1" width="6" height="6" rx="1" />
                    <rect x="9" y="1" width="6" height="6" rx="1" />
                    <rect x="1" y="9" width="6" height="6" rx="1" />
                    <rect x="9" y="9" width="6" height="6" rx="1" />
                  </svg>
                  <svg className="size-4" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
                    <path d="M2 4h12M2 8h12M2 12h12" />
                  </svg>
                </div>
              </div>

              {/* files */}
              <div className="flex-1 p-4">
                <div className="grid grid-cols-2 gap-1 sm:grid-cols-4 md:grid-cols-5">
                  <GridFolder name="Movies" itemCount={12} />
                  <GridFolder name="Music" itemCount={847} />
                  <GridFolder name="Photos" itemCount={3241} />
                  <GridFolder name="Documents" itemCount={56} />
                  <GridFolder name="Backups" itemCount={4} />
                  <GridFile
                    name="README.md"
                    meta="4 KB"
                    onClick={() => setReadmeOpen(true)}
                  />
                </div>
              </div>

              {/* statusbar */}
              <div className="flex items-center justify-between border-t border-border px-4 py-1.5 text-[11px] text-muted-foreground/50">
                <span>5 folders, 1 file</span>
                <span>847 GB available</span>
              </div>
            </div>
          </div>
        </div>
      </main>

      {/* ---- readme modal ---- */}
      {readmeOpen && <ReadmeModal onClose={() => setReadmeOpen(false)} />}
    </div>
  );
}
