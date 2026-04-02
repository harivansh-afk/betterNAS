"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { SignOut } from "@phosphor-icons/react";
import {
  isAuthenticated,
  listExports,
  listNodes,
  issueMountProfile,
  logout,
  getMe,
  type StorageExport,
  type MountProfile,
  type NasNode,
  type User,
  ApiError,
} from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { CopyField } from "../copy-field";

export default function Home() {
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);
  const [nodes, setNodes] = useState<NasNode[]>([]);
  const [exports, setExports] = useState<StorageExport[]>([]);
  const [selectedExportId, setSelectedExportId] = useState<string | null>(null);
  const [mountProfile, setMountProfile] = useState<MountProfile | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!isAuthenticated()) {
      router.replace("/login");
      return;
    }

    async function load() {
      try {
        const [me, registeredNodes, exps] = await Promise.all([
          getMe(),
          listNodes(),
          listExports(),
        ]);
        setUser(me);
        setNodes(registeredNodes);
        setExports(exps);
      } catch (err) {
        if (err instanceof ApiError && err.status === 401) {
          router.replace("/login");
          return;
        }
        setFeedback(err instanceof Error ? err.message : "Failed to load");
      } finally {
        setLoading(false);
      }
    }

    load();
  }, [router]);

  async function handleSelectExport(exportId: string) {
    setSelectedExportId(exportId);
    setMountProfile(null);
    setFeedback(null);

    try {
      const profile = await issueMountProfile(exportId);
      setMountProfile(profile);
    } catch (err) {
      setFeedback(
        err instanceof Error ? err.message : "Failed to issue mount profile",
      );
    }
  }

  async function handleLogout() {
    await logout();
    router.replace("/login");
  }

  if (loading) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-background">
        <p className="text-sm text-muted-foreground">Loading...</p>
      </main>
    );
  }

  const selectedExport = selectedExportId
    ? (exports.find((e) => e.id === selectedExportId) ?? null)
    : null;

  return (
    <main className="min-h-screen bg-background">
      <div className="mx-auto flex max-w-5xl flex-col gap-8 px-4 py-8 sm:px-6">
        {/* header */}
        <div className="flex items-center justify-between">
          <div className="flex flex-col gap-0.5">
            <Link
              href="/"
              className="text-xs text-muted-foreground transition-colors hover:text-foreground"
            >
              betterNAS
            </Link>
            <h1 className="text-xl font-semibold tracking-tight">
              Control Plane
            </h1>
          </div>
          {user && (
            <div className="flex items-center gap-3">
              <Link
                href="/docs"
                className="text-sm text-muted-foreground transition-colors hover:text-foreground"
              >
                Docs
              </Link>
              <span className="text-sm text-muted-foreground">
                {user.username}
              </span>
              <Button variant="ghost" size="sm" onClick={handleLogout}>
                <SignOut className="mr-1 size-4" />
                Sign out
              </Button>
            </div>
          )}
        </div>

        {feedback !== null && (
          <div className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
            {feedback}
          </div>
        )}

        {/* nodes */}
        <section className="flex flex-col gap-4">
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-medium">Nodes</h2>
            <span className="text-xs text-muted-foreground">
              {nodes.filter((n) => n.status === "online").length} online
              {nodes.filter((n) => n.status === "offline").length > 0 &&
                `, ${nodes.filter((n) => n.status === "offline").length} offline`}
            </span>
          </div>

          {nodes.length === 0 ? (
            <p className="py-6 text-center text-sm text-muted-foreground">
              No nodes registered yet. Install and start the node agent on the
              machine that owns your files.
            </p>
          ) : (
            <div className="grid gap-3 sm:grid-cols-2">
              {nodes.map((node) => (
                <div
                  key={node.id}
                  className="flex items-start justify-between gap-4 rounded-lg border p-4"
                >
                  <div className="flex flex-col gap-1 overflow-hidden">
                    <span className="text-sm font-medium">
                      {node.displayName}
                    </span>
                    <span className="truncate text-xs text-muted-foreground">
                      {node.directAddress ?? node.relayAddress ?? node.machineId}
                    </span>
                    <span className="text-xs text-muted-foreground">
                      Last seen {formatTimestamp(node.lastSeenAt)}
                    </span>
                  </div>
                  <Badge
                    variant={node.status === "offline" ? "outline" : "secondary"}
                    className={cn(
                      "shrink-0",
                      node.status === "online" &&
                        "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300",
                      node.status === "degraded" &&
                        "bg-amber-500/15 text-amber-700 dark:text-amber-300",
                    )}
                  >
                    {node.status}
                  </Badge>
                </div>
              ))}
            </div>
          )}
        </section>

        {/* exports + mount */}
        <div className="grid gap-8 lg:grid-cols-[1fr_380px]">
          {/* exports list */}
          <section className="flex flex-col gap-4">
            <h2 className="text-sm font-medium">Exports</h2>

            {exports.length === 0 ? (
              <p className="py-6 text-center text-sm text-muted-foreground">
                {nodes.length === 0
                  ? "No exports yet. Start the node agent to register one."
                  : "No connected exports. Start the node agent or wait for reconnection."}
              </p>
            ) : (
              <div className="flex flex-col gap-2">
                {exports.map((exp) => {
                  const isSelected = exp.id === selectedExportId;

                  return (
                    <button
                      key={exp.id}
                      onClick={() => handleSelectExport(exp.id)}
                      className={cn(
                        "flex items-start justify-between gap-4 rounded-lg border p-4 text-left text-sm transition-colors",
                        isSelected
                          ? "border-foreground/20 bg-muted/50"
                          : "hover:bg-muted/30",
                      )}
                    >
                      <div className="flex flex-col gap-1 overflow-hidden">
                        <span className="font-medium">{exp.label}</span>
                        <span className="truncate text-xs text-muted-foreground">
                          {exp.path}
                        </span>
                      </div>
                      <span className="shrink-0 text-xs text-muted-foreground">
                        {exp.protocols.join(", ")}
                      </span>
                    </button>
                  );
                })}
              </div>
            )}
          </section>

          {/* mount profile */}
          <section className="flex flex-col gap-4">
            <h2 className="text-sm font-medium">
              {selectedExport ? `Mount ${selectedExport.label}` : "Mount"}
            </h2>

            {mountProfile === null ? (
              <p className="py-6 text-center text-sm text-muted-foreground">
                Select an export to see the mount URL and credentials.
              </p>
            ) : (
              <div className="flex flex-col gap-5">
                <CopyField label="Mount URL" value={mountProfile.mountUrl} />
                <CopyField
                  label="Username"
                  value={mountProfile.credential.username}
                />

                <p className="rounded-lg border bg-muted/20 px-3 py-2.5 text-xs leading-relaxed text-muted-foreground">
                  Use your betterNAS account password when Finder prompts. v1
                  does not issue a separate WebDAV password.
                </p>

                <div className="flex flex-col gap-1.5">
                  <h3 className="text-xs font-medium">Finder steps</h3>
                  <ol className="flex flex-col gap-1 text-xs text-muted-foreground">
                    <li>1. Go &gt; Connect to Server in Finder.</li>
                    <li>2. Paste the mount URL.</li>
                    <li>3. Enter your betterNAS username and password.</li>
                    <li>4. Optionally save to Keychain.</li>
                  </ol>
                </div>
              </div>
            )}
          </section>
        </div>
      </div>
    </main>
  );
}

function formatTimestamp(value: string): string {
  const trimmedValue = value.trim();
  if (trimmedValue === "") return "Never";

  const parsed = new Date(trimmedValue);
  if (Number.isNaN(parsed.getTime())) return trimmedValue;

  return parsed.toLocaleString();
}
