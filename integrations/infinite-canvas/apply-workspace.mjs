import { readFile, writeFile, cp } from 'node:fs/promises'
import path from 'node:path'

export async function applyWorkspace(web, replace) {
  await cp(new URL('./simple-workspace.tsx', import.meta.url), path.join(web, 'src/simple-workspace.tsx'))
  await replace('src/stores/use-canvas-side-panel-store.ts', 'return window.innerWidth >= 768 && localStorage.getItem(OPEN_KEY) !== "0";', 'return false;')
  await replace('src/pages/canvas/project.tsx', 'import { CanvasTopBar }', 'import { WorkspaceTopBar as CanvasTopBar, WorkspaceTools as CanvasToolbar, WorkspaceNodeToolbar as CanvasNodeHoverToolbar, WorkspaceZoom as CanvasZoomControls, WorkspaceComposer, WorkspaceEmpty, useSimpleWorkspace } from "@/simple-workspace";\nimport { CanvasTopBar as OriginalCanvasTopBar }')
  await replace('src/pages/canvas/project.tsx', 'import { CanvasZoomControls }', 'import { CanvasZoomControls as OriginalCanvasZoomControls }')
  await replace('src/pages/canvas/project.tsx', 'import { CanvasToolbar }', 'import { CanvasToolbar as OriginalCanvasToolbar }')
  await replace('src/pages/canvas/project.tsx', 'import { CanvasNodeHoverToolbar, CanvasNodeInfoModal }', 'import { CanvasNodeInfoModal }')
  await replace('src/pages/canvas/project.tsx', 'useState<CanvasBackgroundMode>("lines")', 'useState<CanvasBackgroundMode>("dots")')
  await replace('src/stores/canvas/use-canvas-store.ts', 'backgroundMode: "lines",', 'backgroundMode: "dots",')
  await replace('src/pages/canvas/project.tsx', 'showPanel={!isNodeResizing', 'showPanel={node.type !== CanvasNodeType.Image && !isNodeResizing')
  await replace('src/pages/canvas/project.tsx', '<CanvasTopBar\n', '<CanvasTopBar busy={workspace.busy}\n')
  await replace('src/components/canvas/canvas-side-panel.tsx', '<div className="flex items-center gap-5 px-4 pt-3.5">', '<div className="flex items-center gap-5 px-4 pt-3.5">\n                    <button type="button" className="ml-auto order-last text-lg" aria-label={t("canvas.collapsePanel")} onClick={() => useCanvasSidePanelStore.getState().closePanel()}>×</button>')
  await replace('src/pages/canvas/project.tsx', 'setViewport({ x: size.width / 2, y: size.height / 2, k: 1 });', `const items = nodesRef.current;
        if (!items.length) { setViewport({ x: size.width / 2, y: size.height / 2, k: 1 }); return; }
        const left = Math.min(...items.map(node => node.position.x));
        const top = Math.min(...items.map(node => node.position.y));
        const right = Math.max(...items.map(node => node.position.x + node.width));
        const bottom = Math.max(...items.map(node => node.position.y + node.height));
        const availableHeight = Math.max(100, size.height - 320);
        const k = Math.max(.05, Math.min(1.2, (size.width - 110) / Math.max(1, right - left), availableHeight / Math.max(1, bottom - top)));
        setViewport({ k, x: size.width / 2 - (left + right) / 2 * k, y: 65 + availableHeight / 2 - (top + bottom) / 2 * k });`)
  await replace('src/pages/canvas/project.tsx', '    if (!projectLoaded) return <CanvasRefreshShell />;', `    const workspace = useSimpleWorkspace({
        projectId, loaded: projectLoaded, nodes, config: effectiveConfig, runningId: runningNodeId,
        setNodes, setConnections, select: setSelectedNodeIds, center: getCanvasCenter,
        setViewport, size, generate: handleGenerateNode,
        onFailure: (id) => stopGenerationByRunningId(id),
    });

    if (!projectLoaded) return <CanvasRefreshShell />;`)
  await replace('src/pages/canvas/project.tsx', '<main className="flex h-full min-h-0 overflow-hidden"', '<main className="workspace-editor flex h-full min-h-0 overflow-hidden"')
  await replace('src/pages/canvas/project.tsx', '<CanvasNodeHoverToolbar\n', '<CanvasNodeHoverToolbar\n                    onUseReference={(id) => { workspace.addReferences([id]); document.querySelector<HTMLTextAreaElement>(".workspace-composer textarea")?.focus(); }}\n')
  await replace('src/pages/canvas/project.tsx', 'onToggleDialog={(node) => setDialogNodeId((current) => (current === node.id ? null : node.id))}', `onToggleDialog={(node) => {
                        if (node.type !== CanvasNodeType.Image) { setDialogNodeId((current) => current === node.id ? null : node.id); return; }
                        workspace.setDraft(current => ({ ...current, prompt: node.metadata?.prompt || "", references: connections.filter(connection => connection.toNodeId === node.id).map(connection => connection.fromNodeId) }));
                        document.querySelector<HTMLTextAreaElement>(".workspace-composer textarea")?.focus();
                    }}`)
  await replace('src/pages/canvas/project.tsx', '                <CanvasToolbar\n', `                {!nodes.length && <WorkspaceEmpty onUpload={() => handleUploadRequest()} />}
                <WorkspaceComposer key={projectId} workspace={workspace} nodes={nodes} onUpload={() => handleUploadRequest()} onStop={() => { if (runningNodeId) confirmStopGeneration(runningNodeId); }} />
                <CanvasToolbar
`)
  // Show actual upstream failure details on the card instead of a generic batch message.
  await replace('src/pages/canvas/project.tsx', 'errorDetails: hasSuccess ? undefined : t("canvas.projectPage.allFailed")', 'errorDetails: hasSuccess ? undefined : firstError || t("canvas.projectPage.allFailed")')
  const file = path.join(web, 'src/pages/canvas/project.tsx')
  await writeFile(file, (await readFile(file, 'utf8')).replaceAll('project.backgroundMode || "lines"', 'project.backgroundMode || "dots"'))
  const nodeFile = path.join(web, 'src/components/canvas/canvas-node.tsx')
  await writeFile(nodeFile, (await readFile(nodeFile, 'utf8')).replaceAll('title={t("common.download")}', 'data-workspace-inline-download title={t("common.download")}'))
}
