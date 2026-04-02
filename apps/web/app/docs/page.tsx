"use client";

import { useState } from "react";
import Link from "next/link";
import { Check, Copy } from "@phosphor-icons/react";

function CodeBlock({ children, label }: { children: string; label?: string }) {
  const [copied, setCopied] = useState(false);

  return (
    <div className="group relative">
      {label && (
        <span className="mb-1.5 block text-xs text-muted-foreground">
          {label}
        </span>
      )}
      <pre className="overflow-x-auto rounded-lg border bg-muted/40 p-4 pr-12 font-mono text-xs leading-relaxed text-foreground">
        <code>{children}</code>
      </pre>
      <button
        onClick={async () => {
          await navigator.clipboard.writeText(children);
          setCopied(true);
          window.setTimeout(() => setCopied(false), 1500);
        }}
        className="absolute right-2 top-2 flex items-center gap-1 rounded-md border bg-background/80 px-2 py-1 text-xs text-muted-foreground opacity-0 backdrop-blur transition-opacity hover:text-foreground group-hover:opacity-100"
        aria-label="Copy to clipboard"
      >
        {copied ? (
          <>
            <Check size={12} weight="bold" /> Copied
          </>
        ) : (
          <>
            <Copy size={12} /> Copy
          </>
        )}
      </button>
    </div>
  );
}

export default function DocsPage() {
  return (
    <main className="min-h-screen bg-background">
      <div className="mx-auto flex max-w-2xl flex-col gap-10 px-4 py-12 sm:px-6">
        {/* header */}
        <div className="flex flex-col gap-4">
          <div className="flex items-center justify-between">
            <Link
              href="/"
              className="inline-flex items-center gap-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
            >
              <svg className="size-4" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
                <path d="M10 3L5 8l5 5" />
              </svg>
            </Link>
            <Link
              href="/login"
              className="text-sm text-muted-foreground transition-colors hover:text-foreground"
            >
              Sign in
            </Link>
          </div>
          <div className="flex flex-col gap-3">
            <h1 className="text-2xl font-semibold tracking-tight">
              betterNAS
            </h1>
            <p className="text-sm leading-relaxed text-muted-foreground">
              Mount VMs and remote filesystems on your Mac as native Finder
              volumes. No special client, no syncing - just your files, where
              you expect them.
            </p>
            <p className="text-sm leading-relaxed text-muted-foreground">
              Soon: a unified layer across your phone, computer, and AI agents.
              A safe, modular backup of your filesystem that you can use
              natively - and a way to deploy agents on your own infrastructure
              without giving up control.
            </p>
            <h2 className="mt-4 text-lg font-semibold tracking-tight">
              Getting started
            </h2>
            <p className="text-sm leading-relaxed text-muted-foreground">
              One account works everywhere: the web app, the node agent, and
              Finder. Set up the node, confirm it is online, then mount your
              export.
            </p>
          </div>
        </div>

        {/* prerequisites */}
        <section className="flex flex-col gap-3">
          <h2 className="text-sm font-medium">Prerequisites</h2>
          <ul className="flex flex-col gap-1.5 text-sm text-muted-foreground">
            <li>- A betterNAS account</li>
            <li>- A machine with the files you want to expose</li>
            <li>- An export folder on that machine</li>
            <li>
              - A public HTTPS URL that reaches your node directly (for Finder
              mounting)
            </li>
          </ul>
        </section>

        {/* step 1 */}
        <section className="flex flex-col gap-3">
          <h2 className="text-sm font-medium">1. Install the node binary</h2>
          <p className="text-sm text-muted-foreground">
            Run this on the machine that owns the files.
          </p>
          <CodeBlock>
            {`curl -fsSL https://raw.githubusercontent.com/harivansh-afk/betterNAS/main/scripts/install-betternas-node.sh | sh`}
          </CodeBlock>
        </section>

        {/* step 2 */}
        <section className="flex flex-col gap-3">
          <h2 className="text-sm font-medium">2. Start the node</h2>
          <p className="text-sm text-muted-foreground">
            Replace the placeholders with your account, export path, and public
            node URL.
          </p>
          <CodeBlock>
            {`BETTERNAS_CONTROL_PLANE_URL=https://api.betternas.com \\
BETTERNAS_USERNAME=your-username \\
BETTERNAS_PASSWORD='your-password' \\
BETTERNAS_EXPORT_PATH=/absolute/path/to/export \\
BETTERNAS_NODE_DIRECT_ADDRESS=https://your-public-node-url \\
betternas-node`}
          </CodeBlock>
          <div className="flex flex-col gap-1 text-sm text-muted-foreground">
            <p>
              <span className="font-medium text-foreground">Export path</span>{" "}
              - the directory you want to expose through betterNAS.
            </p>
            <p>
              <span className="font-medium text-foreground">
                Direct address
              </span>{" "}
              - the real public HTTPS base URL that reaches your node directly.
            </p>
          </div>
        </section>

        {/* step 3 */}
        <section className="flex flex-col gap-3">
          <h2 className="text-sm font-medium">3. Confirm the node is online</h2>
          <p className="text-sm text-muted-foreground">
            Open the control plane after the node starts. You should see:
          </p>
          <ul className="flex flex-col gap-1.5 text-sm text-muted-foreground">
            <li>- Your node appears as online</li>
            <li>- Your export appears in the exports list</li>
            <li>
              - Issuing a mount profile gives you a WebDAV URL, not an HTML
              login page
            </li>
          </ul>
        </section>

        {/* step 4 */}
        <section className="flex flex-col gap-3">
          <h2 className="text-sm font-medium">4. Mount in Finder</h2>
          <ol className="flex flex-col gap-1.5 text-sm text-muted-foreground">
            <li>1. Open Finder, then Go &gt; Connect to Server.</li>
            <li>
              2. Copy the mount URL from the control plane and paste it in.
            </li>
            <li>
              3. Sign in with the same username and password you used for the
              web app and node agent.
            </li>
            <li>
              4. Save to Keychain only if you want Finder to remember the
              password.
            </li>
          </ol>
        </section>

        {/* note about public urls */}
        <section className="flex flex-col gap-2 rounded-lg border border-border/60 bg-muted/20 px-4 py-3">
          <h2 className="text-sm font-medium">A note on public URLs</h2>
          <p className="text-sm leading-relaxed text-muted-foreground">
            Finder mounting only works when the node URL is directly reachable
            over HTTPS. Avoid gateways that show their own login page before
            forwarding traffic. A good check: load{" "}
            <code className="rounded bg-muted px-1 py-0.5 text-xs">/dav/</code>{" "}
            on your node URL. A working node responds with WebDAV headers, not
            HTML.
          </p>
        </section>
      </div>
    </main>
  );
}
