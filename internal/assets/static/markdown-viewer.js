(() => {
  const mermaidCodeSelector = "pre > code.language-mermaid";
  const mermaidScriptPath = "/assets/vendor/mermaid.min.js";
  let mermaidLoad;

  function loadMermaid() {
    if (window.mermaid) {
      return Promise.resolve(window.mermaid);
    }
    if (mermaidLoad) {
      return mermaidLoad;
    }

    mermaidLoad = new Promise((resolve, reject) => {
      const script = document.createElement("script");
      script.src = mermaidScriptPath;
      script.defer = true;
      script.onload = () => {
        if (window.mermaid) {
          resolve(window.mermaid);
          return;
        }
        reject(new Error("Mermaid loaded without exposing window.mermaid"));
      };
      script.onerror = () => reject(new Error(`Failed to load ${mermaidScriptPath}`));
      document.head.append(script);
    });

    return mermaidLoad;
  }

  function prepareMermaidBlocks(blocks) {
    return blocks.map((code, index) => {
      const pre = code.closest("pre");
      const diagram = document.createElement("div");
      diagram.className = "mermaid mdview-mermaid";
      diagram.dataset.mdviewMermaid = "pending";
      diagram.id = `mdview-mermaid-${index + 1}`;
      diagram.textContent = code.textContent.trim();

      if (pre) {
        pre.replaceWith(diagram);
      } else {
        code.replaceWith(diagram);
      }

      return diagram;
    });
  }

  async function renderMermaid() {
    const blocks = Array.from(document.querySelectorAll(mermaidCodeSelector));
    if (blocks.length === 0) {
      return;
    }

    const mermaid = await loadMermaid();
    mermaid.initialize({
      deterministicIds: true,
      flowchart: {
        htmlLabels: false,
      },
      securityLevel: "strict",
      startOnLoad: false,
      theme: "default",
    });

    const diagrams = prepareMermaidBlocks(blocks);
    await mermaid.run({ nodes: diagrams, suppressErrors: false });
    for (const diagram of diagrams) {
      diagram.dataset.mdviewMermaid = diagram.querySelector("svg") ? "rendered" : "empty";
    }
  }

  document.addEventListener("DOMContentLoaded", () => {
    renderMermaid().catch((error) => {
      document.documentElement.dataset.mdviewMermaid = "error";
      console.error("mdview Mermaid render failed", error);
    });
  });
})();
