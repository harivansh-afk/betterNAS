"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import {
  Globe,
  HardDrives,
  LinkSimple,
  SignOut,
  Warning,
} from "@phosphor-icons/react";
import {
  isAuthenticated,
  listExports,
  issueMountProfile,
  logout,
  getMe,
  type StorageExport,
  type MountProfile,
  type User,
  ApiError,
} from "@/lib/api";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { cn } from "@/lib/utils";
import { CopyField } from "./copy-field";

export default function Home() {
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);
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
        const [me, exps] = await Promise.all([getMe(), listExports()]);
        setUser(me);
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
      <div className="mx-auto flex max-w-6xl flex-col gap-8 px-4 py-8 sm:px-6">
        <div className="flex flex-col gap-4">
          <div className="flex items-start justify-between">
            <div className="flex flex-col gap-1">
              <p className="text-xs font-medium uppercase tracking-widest text-muted-foreground">
                betterNAS
              </p>
              <h1 className="font-heading text-2xl font-semibold tracking-tight">
                Control Plane
              </h1>
            </div>
            {user && (
              <div className="flex items-center gap-3">
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

          <div className="flex flex-wrap items-center gap-2">
            <Badge variant="outline">
              <Globe data-icon="inline-start" />
              {process.env.NEXT_PUBLIC_BETTERNAS_API_URL || "local"}
            </Badge>
            <Badge variant="secondary">
              {exports.length === 1 ? "1 export" : `${exports.length} exports`}
            </Badge>
          </div>

          {user && (
            <Card>
              <CardHeader>
                <CardTitle>Node agent setup</CardTitle>
                <CardDescription>
                  Run the node binary on the machine that owns the files with
                  the same account credentials you use here and in Finder.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="flex flex-col gap-4">
                  <pre className="overflow-x-auto rounded-xl border bg-muted/40 p-4 text-xs text-foreground">
                    <code>
                      {`curl -fsSL https://raw.githubusercontent.com/harivansh-afk/betterNAS/main/scripts/install-betternas-node.sh | sh`}
                    </code>
                  </pre>
                  <pre className="overflow-x-auto rounded-xl border bg-muted/40 p-4 text-xs text-foreground">
                    <code>
                      {`BETTERNAS_USERNAME=${user.username} BETTERNAS_PASSWORD=... BETTERNAS_EXPORT_PATH=/path/to/export BETTERNAS_NODE_DIRECT_ADDRESS=https://your-public-node-url betternas-node`}
                    </code>
                  </pre>
                </div>
              </CardContent>
            </Card>
          )}
        </div>

        {feedback !== null && (
          <Alert variant="destructive">
            <Warning />
            <AlertTitle>Error</AlertTitle>
            <AlertDescription>{feedback}</AlertDescription>
          </Alert>
        )}

        <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_400px]">
          <Card>
            <CardHeader>
              <CardTitle>Exports</CardTitle>
              <CardDescription>
                Storage exports registered with this control plane.
              </CardDescription>
              <CardAction>
                <Badge variant="secondary">
                  {exports.length === 1
                    ? "1 export"
                    : `${exports.length} exports`}
                </Badge>
              </CardAction>
            </CardHeader>
            <CardContent>
              {exports.length === 0 ? (
                <div className="flex flex-col items-center gap-3 rounded-xl border border-dashed py-10 text-center">
                  <HardDrives size={32} className="text-muted-foreground/40" />
                  <p className="text-sm text-muted-foreground">
                    No exports registered yet. Start the node agent and connect
                    it to this control plane.
                  </p>
                </div>
              ) : (
                <div className="flex flex-col gap-2">
                  {exports.map((storageExport) => {
                    const isSelected = storageExport.id === selectedExportId;

                    return (
                      <button
                        key={storageExport.id}
                        onClick={() => handleSelectExport(storageExport.id)}
                        className={cn(
                          "flex flex-col gap-3 rounded-2xl border p-4 text-left text-sm transition-colors",
                          isSelected
                            ? "border-primary/20 bg-primary/5"
                            : "border-border hover:bg-muted/50",
                        )}
                      >
                        <div className="flex items-start justify-between gap-4">
                          <div className="flex flex-col gap-0.5">
                            <span className="font-medium text-foreground">
                              {storageExport.label}
                            </span>
                            <span className="text-xs text-muted-foreground">
                              {storageExport.id}
                            </span>
                          </div>
                          <Badge variant="secondary" className="shrink-0">
                            {storageExport.protocols.join(", ")}
                          </Badge>
                        </div>

                        <dl className="grid grid-cols-2 gap-x-4 gap-y-2">
                          <div>
                            <dt className="mb-0.5 text-xs uppercase tracking-wide text-muted-foreground">
                              Node
                            </dt>
                            <dd className="truncate text-xs text-foreground">
                              {storageExport.nasNodeId}
                            </dd>
                          </div>
                          <div>
                            <dt className="mb-0.5 text-xs uppercase tracking-wide text-muted-foreground">
                              Mount path
                            </dt>
                            <dd className="text-xs text-foreground">
                              {storageExport.mountPath ?? "/dav/"}
                            </dd>
                          </div>
                          <div className="col-span-2">
                            <dt className="mb-0.5 text-xs uppercase tracking-wide text-muted-foreground">
                              Export path
                            </dt>
                            <dd className="truncate text-xs text-foreground">
                              {storageExport.path}
                            </dd>
                          </div>
                        </dl>
                      </button>
                    );
                  })}
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>
                {selectedExport !== null
                  ? `Mount ${selectedExport.label}`
                  : "Mount instructions"}
              </CardTitle>
              <CardDescription>
                {selectedExport !== null
                  ? "WebDAV mount details for Finder."
                  : "Select an export to see the mount URL and account login details."}
              </CardDescription>
            </CardHeader>
            <CardContent>
              {mountProfile === null ? (
                <div className="flex flex-col items-center gap-3 rounded-xl border border-dashed py-10 text-center">
                  <LinkSimple size={32} className="text-muted-foreground/40" />
                  <p className="text-sm text-muted-foreground">
                    Pick an export to see the Finder mount URL and the username
                    to use with your betterNAS account password.
                  </p>
                </div>
              ) : (
                <div className="flex flex-col gap-6">
                  <div className="flex items-center justify-between">
                    <span className="text-xs text-muted-foreground">
                      Issued profile
                    </span>
                    <Badge
                      variant={mountProfile.readonly ? "secondary" : "default"}
                    >
                      {mountProfile.readonly ? "Read-only" : "Read-write"}
                    </Badge>
                  </div>

                  <Separator />

                  <div className="flex flex-col gap-4">
                    <CopyField
                      label="Mount URL"
                      value={mountProfile.mountUrl}
                    />
                    <CopyField
                      label="Username"
                      value={mountProfile.credential.username}
                    />
                    <Alert>
                      <AlertTitle>
                        Use your betterNAS account password
                      </AlertTitle>
                      <AlertDescription>
                        Enter the same password you use to sign in to betterNAS
                        and run the node agent. v1 does not issue a separate
                        WebDAV password.
                      </AlertDescription>
                    </Alert>
                  </div>

                  <Separator />

                  <dl className="grid grid-cols-2 gap-x-4 gap-y-3">
                    <div>
                      <dt className="mb-0.5 text-xs uppercase tracking-wide text-muted-foreground">
                        Mode
                      </dt>
                      <dd className="text-xs text-foreground">
                        {mountProfile.credential.mode}
                      </dd>
                    </div>
                    <div>
                      <dt className="mb-0.5 text-xs uppercase tracking-wide text-muted-foreground">
                        Password source
                      </dt>
                      <dd className="text-xs text-foreground">
                        Your betterNAS account password
                      </dd>
                    </div>
                  </dl>

                  <Separator />

                  <div className="flex flex-col gap-3">
                    <h3 className="text-sm font-medium">Finder steps</h3>
                    <ol className="flex flex-col gap-2">
                      {[
                        "Open Finder and choose Go, then Connect to Server.",
                        "Paste the mount URL into the server address field.",
                        "Enter your betterNAS username and account password when prompted.",
                        "Save to Keychain only if you want Finder to reuse that same account password.",
                      ].map((step, index) => (
                        <li
                          key={index}
                          className="flex gap-2.5 text-sm text-muted-foreground"
                        >
                          <span className="flex size-5 shrink-0 items-center justify-center rounded-full bg-muted text-xs font-medium text-foreground">
                            {index + 1}
                          </span>
                          {step}
                        </li>
                      ))}
                    </ol>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </main>
  );
}
