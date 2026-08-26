const REPO = "https://github.com/sagnikc395/paperfolio";
const DOWNLOAD = `${REPO}/releases/latest`;
const VERSION = "0.2.0";

function Logo({ size = 32 }: { size?: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 40 40"
      fill="none"
      role="img"
      aria-label="Paperfolio"
    >
      <rect width="40" height="40" rx="9" fill="var(--color-wash)" />
      <rect
        x="10.1"
        y="7.1"
        width="19.8"
        height="25.8"
        rx="2.9"
        stroke="currentColor"
        strokeWidth="2.2"
      />
      <path d="M14.6 14.4h10.8" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" opacity="0.3" />
      <rect x="13.7" y="18.2" width="12.6" height="4.4" rx="2.2" fill="var(--color-gold)" />
      <path d="M14.6 27.1h7.4" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" opacity="0.3" />
    </svg>
  );
}

function Eyebrow({ children }: { children: React.ReactNode }) {
  return (
    <p className="mb-6 flex items-center gap-3 text-[13px] tracking-[0.14em] text-muted uppercase">
      <span className="h-[3px] w-6 rounded-full bg-gold" aria-hidden="true" />
      {children}
    </p>
  );
}

/* A quiet stand-in for the app window: a page with one band highlighted and the
   note that band produced. It shows the loop the app is for, and nothing else. */
function AppMock() {
  return (
    <div className="overflow-hidden rounded-xl border border-hairline bg-surface shadow-[0_18px_50px_rgba(60,45,20,0.10)]">
      <div className="flex items-center gap-2 border-b border-hairline bg-ivory/70 px-4 py-3">
        <span className="h-[11px] w-[11px] rounded-full bg-hairline" />
        <span className="h-[11px] w-[11px] rounded-full bg-hairline" />
        <span className="h-[11px] w-[11px] rounded-full bg-hairline" />
        <span className="ml-3 font-display text-[15px] text-muted">
          Attention Is All You Need
        </span>
      </div>
      <div className="grid gap-0 sm:grid-cols-[1.55fr_1fr]">
        <div className="border-b border-hairline p-7 sm:border-r sm:border-b-0">
          <div className="space-y-[10px]" aria-hidden="true">
            <div className="h-[9px] w-[92%] rounded-full bg-hairline" />
            <div className="h-[9px] w-[86%] rounded-full bg-hairline" />
            <div className="h-[9px] w-[64%] rounded-full bg-highlight/60" />
            <div className="h-[9px] w-[78%] rounded-full bg-highlight/60" />
            <div className="h-[9px] w-[89%] rounded-full bg-hairline" />
            <div className="h-[9px] w-[45%] rounded-full bg-hairline" />
            <div className="h-[9px] w-[83%] rounded-full bg-hairline" />
            <div className="h-[9px] w-[71%] rounded-full bg-hairline" />
          </div>
        </div>
        <div className="p-7">
          <p className="text-[12px] tracking-[0.12em] text-muted uppercase">Page 3</p>
          <p className="mt-4 border-l-2 border-gold pl-4 font-display text-[17px] leading-[1.5] text-ink">
            Self-attention lets every position see every other in one step.
          </p>
          <p className="mt-5 text-[14px] leading-[1.65] text-muted">
            Depth stops being the way you buy context. Compare with the RNN
            baseline in §2.
          </p>
        </div>
      </div>
    </div>
  );
}

const things = [
  {
    title: "Highlight while you read",
    body: "Select a line in the PDF to keep it, with a note and a marker if you want one. The sidebar lists every highlight by page; click one to jump back to where it came from.",
  },
  {      title: "Notes in markdown, mirrored to disk",
    body: "Write markdown notes next to the paper. SQLite holds each note, and every note is also written out as a plain .md file you can open in any editor.",
  },
  {
    title: "One paper, one place",
    body: "The PDF, its highlights, its notes and the loose ideas it sparked stay together under the paper. Mark it unread, reading or read; edit the metadata whenever you like.",
  },
];

export default function App() {
  return (
    <div className="min-h-full">
      <header className="mx-auto flex max-w-3xl items-center justify-between px-6 py-7">
        <a href="/" className="flex items-center gap-3 text-ink">
          <Logo size={30} />
          <span className="font-display text-[19px]">Paperfolio</span>
        </a>
        <a
          href={REPO}
          className="text-[15px] text-muted underline decoration-hairline underline-offset-4 transition-colors hover:text-ink-gold hover:decoration-ink-gold"
        >
          GitHub
        </a>
      </header>

      <main className="mx-auto max-w-3xl px-6">
        <section className="pt-16 pb-20 sm:pt-24">
          <h1 className="rise font-display text-[42px] leading-[1.14] font-normal tracking-[-0.015em] text-balance sm:text-[60px]">
            Read the paper.{" "}
            <span className="swipe">Keep what you learned</span> from it.
          </h1>
          <p
            className="rise mt-8 max-w-xl text-[19px] leading-[1.6] text-muted"
            style={{ animationDelay: "0.1s" }}
          >
            Paperfolio is a native Go + Fyne desktop app for research papers. Import a PDF,
            highlight as you go, and write alongside it — all of it in a folder on
            your computer that you can open, copy and back up yourself.
          </p>

          <div
            className="rise mt-10 flex flex-wrap items-center gap-x-6 gap-y-4"
            style={{ animationDelay: "0.18s" }}
          >
            <a
              href={DOWNLOAD}
              className="rounded-full bg-ink-gold px-7 py-3.5 text-[16px] font-medium text-white transition-colors hover:bg-[#75500e]"
            >
              Download Paperfolio
            </a>
            <a
              href={`${REPO}#running-it`}
              className="text-[16px] text-muted underline decoration-hairline underline-offset-4 transition-colors hover:text-ink-gold hover:decoration-ink-gold"
            >
              Build it from source
            </a>
          </div>
          <p
            className="rise mt-5 text-[14px] text-muted"
            style={{ animationDelay: "0.24s" }}
          >
            Version {VERSION} · Go + Fyne · local-first · free and MIT licensed
          </p>
        </section>

        <section className="pb-24">
          <AppMock />
        </section>

        <section className="border-t border-hairline pt-16 pb-20">
          <Eyebrow>What it does</Eyebrow>
          <div className="grid gap-12 sm:grid-cols-3 sm:gap-8">
            {things.map((thing) => (
              <div key={thing.title}>
                <h2 className="font-display text-[22px] leading-[1.3] text-ink">
                  {thing.title}
                </h2>
                <p className="mt-3 text-[15px] leading-[1.65] text-muted">
                  {thing.body}
                </p>
              </div>
            ))}
          </div>
        </section>

        <section className="border-t border-hairline pt-16 pb-20">
          <Eyebrow>Your data</Eyebrow>
          <div className="grid gap-10 sm:grid-cols-[1fr_1fr] sm:items-start sm:gap-14">
            <div>
              <h2 className="font-display text-[28px] leading-[1.25] text-ink">
                A folder you can find on disk.
              </h2>
              <p className="mt-4 text-[16px] leading-[1.65] text-muted">
                No account, no sync, no server — the app talks to SQLite on your
                own disk. Copy the folder to back your library up, copy it onto
                another Mac to move it, delete it to start over.
              </p>
            </div>
            <pre className="overflow-x-auto rounded-lg border border-hairline bg-surface p-5 font-mono text-[13px] leading-[1.85] text-muted">
{`~/Documents/Paperfolio_Data/
├── paperfolio.db   `}<span className="text-ink">{`# SQLite`}</span>{`
├── uploads/        `}<span className="text-ink">{`# your PDFs`}</span>{`
└── notes/          `}<span className="text-ink">{`# markdown mirror`}</span>
            </pre>
          </div>
        </section>

        <section className="border-t border-hairline pt-16 pb-24">
          <Eyebrow>Installing</Eyebrow>
          <h2 className="font-display text-[28px] leading-[1.25] text-ink">
            A native desktop app.
          </h2>
          <p className="mt-4 max-w-xl text-[16px] leading-[1.65] text-muted">
            Download the native executable for your platform and launch it. Paperfolio
            stores your library locally and opens imported PDFs in your operating
            system's default PDF reader.
          </p>
          <a
            href={DOWNLOAD}
            className="mt-8 inline-block rounded-full border border-hairline bg-surface px-7 py-3.5 text-[16px] font-medium text-ink-gold transition-colors hover:border-ink-gold"
          >
            Download the app
          </a>
        </section>
      </main>

      <footer className="border-t border-hairline">
        <div className="mx-auto flex max-w-3xl flex-wrap items-center justify-between gap-4 px-6 py-8 text-[14px] text-muted">
          <span className="flex items-center gap-2.5 text-ink">
            <Logo size={22} />
            Paperfolio
          </span>
          <span>
            MIT licensed ·{" "}
            <a
              href={REPO}
              className="underline decoration-hairline underline-offset-4 transition-colors hover:text-ink-gold hover:decoration-ink-gold"
            >
              Source on GitHub
            </a>
          </span>
        </div>
      </footer>
    </div>
  );
}
