# Wait for the Vite development server before starting the desktop app.
const url = process.env.FRONTEND_DEVSERVER_URL;

if (!url) {
  console.error("FRONTEND_DEVSERVER_URL is not set");
  process.exit(1);
}

const timeoutMs = 30_000;
const deadline = Date.now() + timeoutMs;

while (Date.now() < deadline) {
  try {
    const response = await fetch(url, { signal: AbortSignal.timeout(1_000) });
    if (response.ok) {
      process.exit(0);
    }
  } catch {
    // The frontend server may still be starting.
  }

  await new Promise((resolve) => setTimeout(resolve, 250));
}

console.error(`Vite did not become ready at ${url} within ${timeoutMs} ms`);
process.exit(1);
