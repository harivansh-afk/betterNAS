"use client";

import { Check, Copy } from "@phosphor-icons/react";
import { useState } from "react";
import { Button } from "@/components/ui/button";

export function CopyField({ label, value }: { label: string; value: string }) {
  const [copied, setCopied] = useState(false);

  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-center justify-between gap-2">
        <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
          {label}
        </span>
        <Button
          size="xs"
          variant="outline"
          onClick={async () => {
            await navigator.clipboard.writeText(value);
            setCopied(true);
            window.setTimeout(() => setCopied(false), 1500);
          }}
        >
          {copied ? (
            <Check data-icon="inline-start" weight="bold" />
          ) : (
            <Copy data-icon="inline-start" />
          )}
          {copied ? "Copied" : "Copy"}
        </Button>
      </div>
      <code className="block break-all rounded-lg bg-muted px-3 py-2 font-mono text-xs text-foreground">
        {value}
      </code>
    </div>
  );
}
