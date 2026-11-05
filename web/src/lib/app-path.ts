const base = (import.meta.env?.BASE_URL || "/").replace(/\/$/, "");

/** Browser URLs include the deployment prefix. React Router paths are relative to its basename. */
export function appURL(path: string) {
    if (!base || !path.startsWith("/") || path.startsWith("//") || path === base || path.startsWith(base + "/")) return path;
    return base + path;
}

export function routerPath(path: string) {
    if (base && path === base) return "/";
    return base && path.startsWith(base + "/") ? path.slice(base.length) : path;
}
