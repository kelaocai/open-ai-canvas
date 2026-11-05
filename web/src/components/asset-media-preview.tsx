import { useEffect, useRef, useState, type ReactNode } from "react";

import { CachedResourceImage } from "@/components/cached-resource-image";
import type { Asset } from "@/stores/use-asset-store";
import type { VideoAsset } from "@/stores/use-asset-store";
import { captureVideoPoster } from "@/lib/video-poster";
import { resolveVideoMediaUrl } from "@/services/file-storage";
import { resolveResourceAccessURL } from "@/services/api/resources";
import { getActiveUserScope } from "@/lib/user-scope";

type AssetMediaPreviewProps = {
    asset?: Asset | null;
    alt: string;
    className?: string;
    fallback?: ReactNode;
};

export function AssetMediaPreview({ asset, alt, className = "", fallback = null }: AssetMediaPreviewProps) {
    if (!asset) return fallback;

    if (asset.kind === "video" && (asset.data.storageKey || asset.data.url)) {
        return <VideoAssetCover asset={asset} alt={alt} className={className} fallback={fallback} />;
    }

    const storageKey = asset.kind === "image" ? asset.data.storageKey : undefined;
    const imageUrl = asset.coverUrl || (asset.kind === "image" ? asset.data.dataUrl : "");
    if (!imageUrl && !storageKey) return fallback;
    return <CachedResourceImage storageKey={storageKey} src={imageUrl} alt={alt} loading="lazy" decoding="async" className={className} fallback={fallback} />;
}

// 缓存截图 Blob，避免资产列表与素材选择器重复抽帧；object URL 由各组件释放。
const videoPosters = new Map<string, Promise<Blob | undefined>>();

function VideoAssetCover({ asset, alt, className, fallback }: { asset: VideoAsset; alt: string; className: string; fallback: ReactNode }) {
    const container = useRef<HTMLSpanElement>(null);
    const sourceKey = asset.data.storageKey || asset.data.url;
    const key = `${getActiveUserScope()}:${sourceKey}`;
    const cover = asset.coverUrl && asset.coverUrl !== asset.data.url ? asset.coverUrl : "";
    const [poster, setPoster] = useState<{ key: string; url: string }>();
    useEffect(() => {
        if (cover) return;
        let cancelled = false;
        let objectURL = "";
        const observer = new IntersectionObserver((entries) => {
            if (!entries.some((entry) => entry.isIntersecting)) return;
            observer.disconnect();
            let pending = videoPosters.get(key);
            if (!pending) {
                pending = resolveVideoMediaUrl(asset.data.storageKey, resolveResourceAccessURL(asset.data.url))
                    .then((url) => captureVideoPoster(url, { maxWidth: 640 }))
                    .then((result) => result.poster)
                    .catch(() => undefined);
                videoPosters.set(key, pending);
                if (videoPosters.size > 60) videoPosters.delete(videoPosters.keys().next().value!);
                void pending.then((blob) => {
                    if (!blob && videoPosters.get(key) === pending) videoPosters.delete(key);
                });
            }
            void pending.then((blob) => {
                if (cancelled || !blob) return;
                objectURL = URL.createObjectURL(blob);
                setPoster({ key, url: objectURL });
            });
        });
        if (container.current) observer.observe(container.current);
        return () => {
            cancelled = true;
            observer.disconnect();
            if (objectURL) URL.revokeObjectURL(objectURL);
        };
    }, [key, cover, asset.data.storageKey, asset.data.url]);
    const url = cover ? resolveResourceAccessURL(cover) : poster?.key === key ? poster.url : "";
    return (
        <span ref={container} className="block h-full w-full">
            {url ? <CachedResourceImage src={url} alt={alt} className={className} loading="lazy" fallback={fallback} /> : fallback}
        </span>
    );
}
