import {
  Globe,
  HardDrives,
  LinkSimple,
  Warning,
} from "@phosphor-icons/react/dist/ssr";
import {
  ControlPlaneConfigurationError,
  ControlPlaneRequestError,
  getControlPlaneConfig,
  issueMountProfile,
  listExports,
  type MountProfile,
  type StorageExport,
} from "@/lib/control-plane";
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
import { Separator } from "@/components/ui/separator";
import { cn } from "@/lib/utils";
import { CopyField } from "./copy-field";

export const dynamic = "force-dynamic";

interface PageProps {
  searchParams: Promise<{ exportId?: string | string[] }>;
}

export default async function Home({ searchParams }: PageProps) {
  const resolvedSearchParams = await searchParams;
  const selectedExportId = readSearchParam(resolvedSearchParams.exportId);
  const controlPlaneConfig = await getControlPlaneConfig();

  let exports: StorageExport[] = [];
  let mountProfile: MountProfile | null = null;
  let feedback: string | null = null;

  try {
    exports = await listExports();

    if (selectedExportId !== null) {
      if (exports.some((e) => e.id === selectedExportId)) {
        mountProfile = await issueMountProfile(selectedExportId);
      } else {
        feedback = `Export "${selectedExportId}" was not found.`;
      }
    }
  } catch (error) {
    if (
      error instanceof ControlPlaneConfigurationError ||
      error instanceof ControlPlaneRequestError
    ) {
      feedback = error.message;
    } else {
      throw error;
    }
  }

  const selectedExport =
    selectedExportId === null
      ? null
      : (exports.find((e) => e.id === selectedExportId) ?? null);

  return (
    <main className="min-h-screen bg-background">
      <div className="mx-auto flex max-w-6xl flex-col gap-8 px-4 py-8 sm:px-6">
        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-1">
            <p className="text-xs font-medium uppercase tracking-widest text-muted-foreground">
              betterNAS
            </p>
            <h1 className="font-heading text-2xl font-semibold tracking-tight">
              Control Plane
            </h1>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <Badge variant="outline">
              <Globe data-icon="inline-start" />
              {controlPlaneConfig.baseUrl ?? "Not configured"}
            </Badge>
            <Badge
              variant={
                controlPlaneConfig.clientToken !== null
                  ? "secondary"
                  : "destructive"
              }
            >
              {controlPlaneConfig.clientToken !== null
                ? "Bearer auth"
                : "No token"}
            </Badge>
            <Badge variant="secondary">
              {exports.length === 1 ? "1 export" : `${exports.length} exports`}
            </Badge>
          </div>
        </div>

        {feedback !== null && (
          <Alert variant="destructive">
            <Warning />
            <AlertTitle>Configuration error</AlertTitle>
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
                      <a
                        key={storageExport.id}
                        href={`/?exportId=${encodeURIComponent(storageExport.id)}`}
                        className={cn(
                          "flex flex-col gap-3 rounded-2xl border p-4 text-sm transition-colors",
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
                      </a>
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
                  ? "Issued WebDAV credentials for Finder."
                  : "Select an export to issue mount credentials."}
              </CardDescription>
            </CardHeader>
            <CardContent>
              {mountProfile === null ? (
                <div className="flex flex-col items-center gap-3 rounded-xl border border-dashed py-10 text-center">
                  <LinkSimple size={32} className="text-muted-foreground/40" />
                  <p className="text-sm text-muted-foreground">
                    Pick an export to issue WebDAV credentials for Finder.
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
                    <CopyField
                      label="Password"
                      value={mountProfile.credential.password}
                    />
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
                        Expires
                      </dt>
                      <dd className="text-xs text-foreground">
                        {mountProfile.credential.expiresAt}
                      </dd>
                    </div>
                  </dl>

                  <Separator />

                  <div className="flex flex-col gap-3">
                    <h3 className="text-sm font-medium">Finder steps</h3>
                    <ol className="flex flex-col gap-2">
                      {[
                        "Open Finder and choose Go, then Connect to Server.",
                        `Paste the mount URL into the server address field.`,
                        "Enter the issued username and password when prompted.",
                        "Save to Keychain only if the credential expiry suits your workflow.",
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

function readSearchParam(value: string | string[] | undefined): string | null {
  if (typeof value === "string" && value.trim() !== "") {
    return value.trim();
  }
  if (Array.isArray(value)) {
    const first = value.find((v) => v.trim() !== "");
    return first?.trim() ?? null;
  }
  return null;
}
