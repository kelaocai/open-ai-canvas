import { readdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";

const base = (process.argv[2] || "/").replace(/\/$/, "");
const directory = process.argv[3] || "web/dist";
if (base && !/^\/[a-zA-Z0-9/_-]+$/.test(base)) throw new Error("Invalid deployment prefix");
const roots = /(?<=[`"'(=])\/(logo\.svg|canvas\/models\/|(?:commerce|doll|ecommerce-models|icons|images|inspiration-thumbs|inspirations|lighting-presets|live2d|mediapipe|previs|short-drama-styles|three)\/|welcome\/(?=credits\.html|workbench-|charge\/|spring\/|wing-it\/))/g;
async function visit(directory) {
    for (const item of await readdir(directory, { withFileTypes: true })) {
        const file = path.join(directory, item.name);
        if (item.isDirectory()) await visit(file);
        else if (/\.(js|css|json|html)$/.test(file)) {
            const original = await readFile(file, "utf8");
            const updated = original.replace(roots, (_, root) => base + "/" + root);
            if (updated !== original) await writeFile(file, updated);
        }
    }
}
if (base) await visit(directory);
