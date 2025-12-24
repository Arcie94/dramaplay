export async function GET() {
    const domain = "https://dramaplay.online";
    const robots = `User-agent: *
Allow: /
Sitemap: ${domain}/sitemap.xml`;

    return new Response(robots, {
        headers: { 'Content-Type': 'text/plain' }
    });
}
