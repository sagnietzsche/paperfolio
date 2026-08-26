package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var latexTemplates = map[string]string{
	"article": `\documentclass[11pt,a4paper]{article}
\usepackage[utf8]{inputenc}
\usepackage{amsmath,amssymb,amsthm}
\usepackage{graphicx}
\usepackage{hyperref}
\usepackage{geometry}
\geometry{margin=1in}

\title{%s}
\author{Your Name}
\date{\today}

\begin{document}
\maketitle

\begin{abstract}
Your abstract here. Summarize the contribution, methods, and key findings in 150--250 words.
\end{abstract}

\section{Introduction}
Introduce the problem and motivation. Cite related work \cite{example2024}.

\section{Related Work}
Discuss prior approaches.

\section{Method}
Describe your method formally. Example equation:
\begin{equation}
  \mathcal{L} = -\sum_{i=1}^{N} y_i \log \hat{y}_i
  \label{eq:loss}
\end{equation}

\section{Experiments}
\begin{table}[h]
\centering
\begin{tabular}{lcc}
\hline
Method & Accuracy & F1 \\
\hline
Baseline & 0.85 & 0.83 \\
Ours & \textbf{0.92} & \textbf{0.90} \\
\hline
\end{tabular}
\caption{Results on benchmark dataset.}
\label{tab:results}
\end{table}

\section{Conclusion}
Summarize contributions and future work.

\bibliographystyle{plain}
\begin{thebibliography}{9}
\bibitem{example2024} Author, Title, Venue, 2024.
\end{thebibliography}

\end{document}
`,
	"ieee": `\documentclass[conference]{IEEEtran}
\usepackage{amsmath,amssymb}
\usepackage{graphicx}
\usepackage{cite}
\usepackage{hyperref}

\title{%s}
\author{\IEEEauthorblockN{Your Name}
\IEEEauthorblockA{\textit{Your Institution} \\ \textit{your.email@example.com}}
}

\begin{document}
\maketitle

\begin{abstract}
Abstract for IEEE conference format.
\end{abstract}

\begin{IEEEkeywords}
keyword1, keyword2, keyword3
\end{IEEEkeywords}

\section{Introduction}
Content here.

\section{Proposed Approach}
\begin{equation}
  y = Wx + b
\end{equation}

\section{Evaluation}
Results and discussion.

\section{Conclusion}
Concluding remarks.

\begin{thebibliography}{00}
\bibitem{b1} Reference 1.
\end{thebibliography}

\end{document}
`,
	"report": `\documentclass[12pt,a4paper]{report}
\usepackage{amsmath,amssymb}
\usepackage{graphicx}
\usepackage{hyperref}
\usepackage{geometry}
\geometry{margin=1in}

\title{%s}
\author{Your Name}
\date{\today}

\begin{document}
\maketitle
\tableofcontents
\newpage

\chapter{Introduction}
Introduction text.

\chapter{Background}
Background text.

\chapter{Methodology}
\begin{equation}
  E = mc^2
\end{equation}

\chapter{Results}
Results.

\chapter{Conclusion}
Conclusion.

\end{document}
`,
	"beamer": `\documentclass{beamer}
\usetheme{Madrid}
\usepackage{amsmath}

\title{%s}
\author{Your Name}
\date{\today}

\begin{document}

\begin{frame}
\titlepage
\end{frame}

\begin{frame}{Outline}
\tableofcontents
\end{frame}

\section{Motivation}
\begin{frame}{Motivation}
  \begin{itemize}
    \item Problem statement
    \item Why it matters
    \item Our contribution
  \end{itemize}
\end{frame}

\section{Method}
\begin{frame}{Method}
  \begin{equation}
    \mathbf{h} = \sigma(W\mathbf{x} + \mathbf{b})
  \end{equation}
\end{frame}

\section{Results}
\begin{frame}{Results}
  \begin{itemize}
    \item Key finding 1
    \item Key finding 2
  \end{itemize}
\end{frame}

\begin{frame}{Conclusion}
  Thank you!
\end{frame}

\end{document}
`,
	"letter": `\documentclass[11pt]{letter}
\usepackage{hyperref}
\signature{Your Name \\ Your Title \\ Institution}
\address{Your Address \\ City, Country \\ Email}

\begin{document}
\begin{letter}{Recipient Name \\ Institution \\ Address}
\opening{Dear Sir/Madam,}

This is a letter regarding "%s".

Content of the letter goes here.

\closing{Sincerely,}
\end{letter}
\end{document}
`,
	"minimal": `\documentclass{article}
\begin{document}
\title{%s}
\maketitle
Hello, write your \LaTeX\ here. Inline math $E=mc^2$ and display:
\[
  \int_0^\infty e^{-x^2} dx = \frac{\sqrt{\pi}}{2}
\]
\end{document}
`,
}

var latexTemplateNames = []string{"article", "ieee", "report", "beamer", "letter", "minimal"}

func latexTemplateContent(template, title string) string {
	tmpl, ok := latexTemplates[template]
	if !ok {
		tmpl = latexTemplates["article"]
	}
	safeTitle := strings.ReplaceAll(title, "{", "\\{")
	safeTitle = strings.ReplaceAll(safeTitle, "}", "\\}")
	safeTitle = strings.ReplaceAll(safeTitle, "%", "\\%")
	return fmt.Sprintf(tmpl, safeTitle)
}

func writingDir(dataDir string) string {
	return filepath.Join(dataDir, "writing")
}

func latexFilePath(dataDir string, doc LatexDocument) string {
	return filepath.Join(writingDir(dataDir), stamped(doc.Title, doc.ID)+".tex")
}

func writeLatexFile(dataDir string, doc LatexDocument) error {
	dir := writingDir(dataDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	target := latexFilePath(dataDir, doc)
	removeOtherLatexFiles(dir, doc.ID, target)
	return os.WriteFile(target, []byte(doc.Content), 0o644)
}

func removeOtherLatexFiles(dir, docID, keep string) {
	short := docID
	if len(short) > 8 {
		short = short[:8]
	}
	suffix := "-" + short + ".tex"
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if path != keep && strings.HasSuffix(entry.Name(), suffix) {
			_ = os.Remove(path)
		}
	}
}

func (s *Store) mirrorLatexDocument(doc LatexDocument) {
	_ = writeLatexFile(s.DataDir, doc)
}

func (s *Store) removeLatexDocumentFiles(doc LatexDocument) {
	dir := writingDir(s.DataDir)
	removeOtherLatexFiles(dir, doc.ID, "")
	_ = os.Remove(latexFilePath(s.DataDir, doc))
	if entries, err := os.ReadDir(dir); err == nil && len(entries) == 0 {
		_ = os.Remove(dir)
	}
}

var latexSnippets = map[string]string{
	"Section":      "\\section{Section Title}\nContent here.\n",
	"Subsection":   "\\subsection{Subsection Title}\nContent here.\n",
	"Equation":     "\\begin{equation}\n  E = mc^2 \\label{eq:example}\n\\end{equation}\n",
	"Align":        "\\begin{align}\n  a &= b + c \\\\\n  d &= e + f\n\\end{align}\n",
	"Figure":       "\\begin{figure}[h]\n  \\centering\n  \\includegraphics[width=0.8\\linewidth]{figure.pdf}\n  \\caption{Caption.}\n  \\label{fig:example}\n\\end{figure}\n",
	"Table":        "\\begin{table}[h]\n  \\centering\n  \\begin{tabular}{lcc}\n    \\hline\n    A & B & C \\\\\n    \\hline\n    1 & 2 & 3 \\\\\n    \\hline\n  \\end{tabular}\n  \\caption{Caption.}\n  \\label{tab:example}\n\\end{table}\n",
	"Citation":     "\\cite{key}",
	"Itemize":      "\\begin{itemize}\n  \\item First item\n  \\item Second item\n\\end{itemize}\n",
	"Enumerate":    "\\begin{enumerate}\n  \\item First item\n  \\item Second item\n\\end{enumerate}\n",
	"Abstract":     "\\begin{abstract}\nAbstract text here.\n\\end{abstract}\n",
	"Footnote":     "\\footnote{Footnote text.}",
	"URL":          "\\url{https://example.com}",
	"Code Listing": "\\begin{verbatim}\ncode here\n\\end{verbatim}\n",
}

func latexSnippetNames() []string {
	return []string{"Section", "Subsection", "Equation", "Align", "Figure", "Table", "Citation", "Itemize", "Enumerate", "Abstract", "Footnote", "URL", "Code Listing"}
}
