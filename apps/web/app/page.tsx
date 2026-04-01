import { Card } from "@betternas/ui/card";
import styles from "./page.module.css";

const lanes = [
  {
    title: "NAS node",
    body: "Runs on the storage machine. Exposes WebDAV, reports exports, and stays close to the bytes.",
  },
  {
    title: "Control plane",
    body: "Owns users, devices, nodes, grants, mount profiles, and cloud profiles.",
  },
  {
    title: "Local device",
    body: "Consumes mount profiles and uses Finder WebDAV flows before we ship a helper app.",
  },
  {
    title: "Cloud layer",
    body: "Keeps Nextcloud optional and thin for browser, mobile, and sharing flows.",
  },
];

export default function Home() {
  return (
    <main className={styles.page}>
      <section className={styles.hero}>
        <p className={styles.eyebrow}>betterNAS monorepo</p>
        <h1 className={styles.title}>
          Contract-first scaffold for NAS mounts and cloud mode.
        </h1>
        <p className={styles.copy}>
          The repo is organized so each system part can be built in parallel
          without inventing new interfaces. The source of truth is the root
          contract plus the shared contracts package.
        </p>
      </section>

      <section className={styles.grid}>
        {lanes.map((lane) => (
          <Card
            key={lane.title}
            className={styles.card}
            title={lane.title}
            href="/#"
          >
            {lane.body}
          </Card>
        ))}
      </section>
    </main>
  );
}
