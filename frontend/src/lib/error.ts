export function handleError(err: any, fallback: string): string {
  if (err instanceof Error && err.message.trim() !== "") {
    return err.message;
  }
  if (err && typeof err.message === "string" && err.message.trim() !== "") {
    return err.message;
  }
  if (typeof err === "string" && err.trim() !== "") {
    return err;
  }
  return fallback;
}
