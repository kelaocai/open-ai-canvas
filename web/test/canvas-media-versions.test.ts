import { describe, expect, test } from "bun:test";

import { adoptMediaVersionToNodes, createNodeFromMediaVersion, hasMediaVersionHistory, mediaVersionPreviewNode } from "@/lib/canvas/canvas-media-versions";
import { commitCanvasGenerationResult } from "@/lib/canvas/canvas-generation-result";
import { CanvasNodeType, type CanvasNodeData } from "@/types/canvas";

function batchRoot(): CanvasNodeData {
    return {
        id: "root",
        type: CanvasNodeType.Image,
        title: "画面",
        position: { x: 50, y: 80 },
        width: 720,
        height: 480,
        metadata: {
            content: "/banana.png",
            isBatchRoot: true,
            batchChildIds: ["child-1", "child-2"],
            primaryImageId: "child-1",
            imageBatchExpanded: false,
            taskId: "banana-task",
            status: "success",
            currentMediaVersionId: "gpt-v1",
            mediaVersions: [{ id: "gpt-v1", number: 1, createdAt: "2026-11-07T08:19:00Z", source: "generate", media: { content: "/gpt.png", naturalWidth: 1536, naturalHeight: 1024 }, generation: { model: "gpt-image-2" } }],
        },
    };
}

describe("media history on image batches", () => {
    test("keeps the history entry visible on batch roots and children", () => {
        const root = batchRoot();
        expect(hasMediaVersionHistory(root)).toBe(true);
        expect(hasMediaVersionHistory({ ...root, metadata: { ...root.metadata, isBatchRoot: false, batchRootId: "parent" } })).toBe(true);
        expect(hasMediaVersionHistory({ ...root, metadata: { ...root.metadata, isBatchRoot: false, content: "/gpt.png" } })).toBe(true);
    });

    test("records the current batch preview alongside the earlier image", () => {
        const root = batchRoot();
        const [next] = commitCanvasGenerationResult([root], root, root, "banana-task");
        expect(next.metadata?.mediaVersions?.map((version) => version.media.content)).toEqual(["/gpt.png", "/banana.png"]);
        expect(hasMediaVersionHistory(next)).toBe(true);
    });

    test("extracts and previews a plain image without inheriting batch bindings", () => {
        const root = batchRoot();
        const version = root.metadata!.mediaVersions![0];
        for (const node of [createNodeFromMediaVersion(root, version.id, "extracted")!, mediaVersionPreviewNode(root, version)]) {
            expect(node.metadata?.content).toBe("/gpt.png");
            expect(node.metadata?.isBatchRoot).toBeUndefined();
            expect(node.metadata?.batchChildIds).toBeUndefined();
            expect(node.metadata?.primaryImageId).toBeUndefined();
        }
    });

    test("restores the old image and detaches existing children without deleting their media", () => {
        const root = batchRoot();
        const children = ["child-1", "child-2"].map((id) => ({ ...root, id, metadata: { content: `/${id}.png`, batchRootId: root.id } }));
        const result = adoptMediaVersionToNodes([root, ...children], root.id, "gpt-v1")!;
        expect(result).toHaveLength(3);
        expect(result[0].metadata?.content).toBe("/gpt.png");
        expect(result[0].metadata?.isBatchRoot).toBeUndefined();
        expect(result[0].metadata?.mediaVersions?.some((version) => version.media.content === "/banana.png")).toBe(true);
        expect(result.slice(1).map((node) => node.metadata?.content)).toEqual(["/child-1.png", "/child-2.png"]);
        expect(result.slice(1).every((node) => !node.metadata?.batchRootId)).toBe(true);
        expect(root.metadata?.isBatchRoot).toBe(true);
    });
});
