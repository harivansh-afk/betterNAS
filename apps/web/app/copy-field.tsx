"use client";

import { Button } from "@betternas/ui/button";
import { Code } from "@betternas/ui/code";
import { useState } from "react";
import styles from "./copy-field.module.css";

export function CopyField({ label, value }: { label: string; value: string }) {
  const [copied, setCopied] = useState(false);

  return (
    <div className={styles.field}>
      <div className={styles.header}>
        <span className={styles.label}>{label}</span>
        <Button
          className={styles.button}
          onClick={async () => {
            await navigator.clipboard.writeText(value);
            setCopied(true);
            window.setTimeout(() => setCopied(false), 1500);
          }}
        >
          {copied ? "Copied" : "Copy"}
        </Button>
      </div>
      <Code className={styles.value}>{value}</Code>
    </div>
  );
}
