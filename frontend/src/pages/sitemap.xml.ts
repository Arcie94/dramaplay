import type { APIRoute } from "astro";

export const GET: APIRoute = async () => {
  const siteUrl = "https://dramaplay.online";
  const staticPages = ["", "/trending", "/mylist", "/search", "/history"];

  // Optional: Fetch Dynamic Routes (e.g. Top 10 Trending)
  // We skip this for now to keep it fast, or we can fetch from internal API?
  // Let's stick to static for reliability first.

  const sitemap = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  ${staticPages
    .map((path) => {
      return `
  <url>
    <loc>${siteUrl}${path}</loc>
    <lastmod>${new Date().toISOString()}</lastmod>
    <changefreq>daily</changefreq>
    <priority>${path === "" ? "1.0" : "0.8"}</priority>
  </url>`;
    })
    .join("")}
</urlset>`;

  return new Response(sitemap, {
    headers: {
      "Content-Type": "application/xml",
    },
  });
};
