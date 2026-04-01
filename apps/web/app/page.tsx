import { Code } from "@betternas/ui/code";
import { CopyField } from "./copy-field";
import {
  ControlPlaneConfigurationError,
  ControlPlaneRequestError,
  getControlPlaneConfig,
  issueMountProfile,
  listExports,
  type MountProfile,
  type StorageExport,
} from "../lib/control-plane";
import styles from "./page.module.css";

export const dynamic = "force-dynamic";

interface PageProps {
  searchParams: Promise<{
    exportId?: string | string[];
  }>;
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
      if (
        exports.some((storageExport) => storageExport.id === selectedExportId)
      ) {
        mountProfile = await issueMountProfile(selectedExportId);
      } else {
        feedback = `Export ${selectedExportId} was not found in the current control-plane response.`;
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
      : (exports.find(
          (storageExport) => storageExport.id === selectedExportId,
        ) ?? null);

  return (
    <main className={styles.page}>
      <section className={styles.hero}>
        <p className={styles.eyebrow}>betterNAS control plane</p>
        <h1 className={styles.title}>
          Mount exports from the live control-plane.
        </h1>
        <p className={styles.copy}>
          This page reads the running control-plane, lists available exports,
          and issues Finder-friendly WebDAV mount credentials for the export you
          select.
        </p>
        <dl className={styles.heroMeta}>
          <div>
            <dt>Control-plane URL</dt>
            <dd>{controlPlaneConfig.baseUrl ?? "Not configured"}</dd>
          </div>
          <div>
            <dt>Auth mode</dt>
            <dd>
              {controlPlaneConfig.clientToken === null
                ? "Missing client token"
                : "Server-side bearer token"}
            </dd>
          </div>
          <div>
            <dt>Exports discovered</dt>
            <dd>{exports.length}</dd>
          </div>
        </dl>
      </section>

      <section className={styles.layout}>
        <section className={styles.panel}>
          <div className={styles.panelHeader}>
            <div>
              <p className={styles.sectionEyebrow}>Exports</p>
              <h2 className={styles.sectionTitle}>
                Registered storage exports
              </h2>
            </div>
            <span className={styles.sectionMeta}>
              {exports.length === 1 ? "1 export" : `${exports.length} exports`}
            </span>
          </div>

          {feedback !== null ? (
            <div className={styles.notice}>{feedback}</div>
          ) : null}

          {exports.length === 0 ? (
            <div className={styles.emptyState}>
              No exports are registered yet. Start the node agent and verify the
              control-plane connection first.
            </div>
          ) : (
            <div className={styles.exportList}>
              {exports.map((storageExport) => {
                const isSelected = storageExport.id === selectedExportId;

                return (
                  <a
                    key={storageExport.id}
                    className={
                      isSelected ? styles.exportCardSelected : styles.exportCard
                    }
                    href={`/?exportId=${encodeURIComponent(storageExport.id)}`}
                  >
                    <div className={styles.exportCardTop}>
                      <div>
                        <h3 className={styles.exportTitle}>
                          {storageExport.label}
                        </h3>
                        <p className={styles.exportId}>{storageExport.id}</p>
                      </div>
                      <span className={styles.exportProtocol}>
                        {storageExport.protocols.join(", ")}
                      </span>
                    </div>

                    <dl className={styles.exportFacts}>
                      <div>
                        <dt>Node</dt>
                        <dd>{storageExport.nasNodeId}</dd>
                      </div>
                      <div>
                        <dt>Mount path</dt>
                        <dd>{storageExport.mountPath ?? "/dav/"}</dd>
                      </div>
                      <div className={styles.exportFactWide}>
                        <dt>Export path</dt>
                        <dd>{storageExport.path}</dd>
                      </div>
                    </dl>
                  </a>
                );
              })}
            </div>
          )}
        </section>

        <aside className={styles.panel}>
          <div className={styles.panelHeader}>
            <div>
              <p className={styles.sectionEyebrow}>Mount instructions</p>
              <h2 className={styles.sectionTitle}>
                {selectedExport === null
                  ? "Select an export"
                  : `Mount ${selectedExport.label}`}
              </h2>
            </div>
          </div>

          {mountProfile === null ? (
            <div className={styles.emptyState}>
              Pick an export to issue a WebDAV mount profile and reveal the URL,
              username, password, and expiry.
            </div>
          ) : (
            <div className={styles.mountPanel}>
              <div className={styles.mountStatus}>
                <div>
                  <p className={styles.sectionEyebrow}>Issued profile</p>
                  <h3 className={styles.mountTitle}>
                    {mountProfile.displayName}
                  </h3>
                </div>
                <span className={styles.mountBadge}>
                  {mountProfile.readonly ? "Read-only" : "Read-write"}
                </span>
              </div>

              <div className={styles.copyFields}>
                <CopyField label="Mount URL" value={mountProfile.mountUrl} />
                <CopyField
                  label="Username"
                  value={mountProfile.credential.username}
                />
                <CopyField
                  label="Password"
                  value={mountProfile.credential.password}
                />
              </div>

              <dl className={styles.mountMeta}>
                <div>
                  <dt>Credential mode</dt>
                  <dd>{mountProfile.credential.mode}</dd>
                </div>
                <div>
                  <dt>Expires at</dt>
                  <dd>{mountProfile.credential.expiresAt}</dd>
                </div>
              </dl>

              <div className={styles.instructions}>
                <h3 className={styles.instructionsTitle}>Finder steps</h3>
                <ol className={styles.instructionsList}>
                  <li>Open Finder and choose Go, then Connect to Server.</li>
                  <li>
                    Paste{" "}
                    <Code className={styles.inlineCode}>
                      {mountProfile.mountUrl}
                    </Code>
                    .
                  </li>
                  <li>When prompted, use the issued username and password.</li>
                  <li>
                    Save credentials in Keychain only if the expiry fits your
                    workflow.
                  </li>
                </ol>
              </div>
            </div>
          )}
        </aside>
      </section>
    </main>
  );
}

function readSearchParam(value: string | string[] | undefined): string | null {
  if (typeof value === "string" && value.trim() !== "") {
    return value.trim();
  }
  if (Array.isArray(value)) {
    const firstValue = value.find(
      (candidateValue) => candidateValue.trim() !== "",
    );
    return firstValue?.trim() ?? null;
  }

  return null;
}
