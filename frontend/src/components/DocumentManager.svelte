<script lang="ts">
  import { auth } from "../stores/auth.svelte.js";
  import { DocumentService, SystemSettingsService } from "@bindings/services/index.js";
  import { DocumentType, type Document } from "@bindings/domain/models.js";
  import { Dialogs } from "@wailsio/runtime";
  import Modal from "./ui/Modal.svelte";
  import FormField from "./ui/FormField.svelte";
  import Input from "./ui/Input.svelte";
  import EmptyState from "./ui/EmptyState.svelte";
  import ConfirmModal from "./ui/ConfirmModal.svelte";
  import * as m from "../paraglide/messages.js";
  import { handleError } from "../lib/error.js";

  // With a patientId this lists and uploads that patient's documents; without one it manages
  // clinic-wide documents.
  let {
    openUploadModal = $bindable(),
    patientId = "",
  }: {
    openUploadModal?: () => void;
    patientId?: string;
  } = $props();

  let documents = $state<Document[]>([]);
  let isLoading = $state(true);
  let exportSuccessMsg = $state("");
  let actionError = $state("");
  let showConfirmDelete = $state(false);
  let docToDelete = $state("");

  // Modal State
  let showUploadModal = $state(false);
  let docName = $state("");
  let docDesc = $state("");
  let selectedFile = $state<File | null>(null);
  let isUploading = $state(false);
  let uploadError = $state("");

  async function loadDocuments() {
    const currentToken = auth.token;
    const currentPatientId = patientId;
    if (!currentToken) {
      isLoading = false;
      return;
    }
    isLoading = true;
    try {
      const result = currentPatientId
        ? await DocumentService.ListPatientDocuments(currentToken, {
            patient_id: currentPatientId,
          })
        : await DocumentService.ListClinicDocuments(currentToken);
      if (auth.token === currentToken && patientId === currentPatientId) {
        documents = result || [];
      }
    } catch (err) {
      if (auth.token === currentToken) {
        console.error("Failed to load documents:", err);
      }
    } finally {
      if (auth.token === currentToken) {
        isLoading = false;
      }
    }
  }

  $effect(() => {
    void patientId;
    if (auth.token) {
      loadDocuments();
    } else {
      documents = [];
      isLoading = false;
    }
  });

  openUploadModal = () => {
    docName = "";
    docDesc = "";
    selectedFile = null;
    uploadError = "";
    showUploadModal = true;
  };

  function handleFileSelect(e: Event) {
    const target = e.target as HTMLInputElement;
    if (target.files && target.files.length > 0) {
      selectedFile = target.files[0];
      if (!docName) {
        docName = selectedFile.name;
      }
    }
  }

  async function handleUpload(e: Event) {
    e.preventDefault();
    if (!selectedFile || !docName) return;
    // Snapshot the owner so a patient switch while the file is read cannot reassign it.
    const targetPatientId = patientId;

    isUploading = true;
    uploadError = "";

    try {
      const reader = new FileReader();
      reader.onload = async (ev) => {
        const result = ev.target?.result as string;
        // Result is data:application/pdf;base64,...
        const base64Data = result.split(",")[1];
        if (!base64Data) {
          uploadError = m.doc_err_parse();
          isUploading = false;
          return;
        }

        try {
          const lowerName = selectedFile?.name.toLowerCase() || "";
          const isDicom =
            lowerName.endsWith(".dcm") ||
            lowerName.endsWith(".dicom") ||
            !!selectedFile?.type.toLowerCase().includes("dicom");
          const mime = isDicom ? "application/dicom" : selectedFile?.type || "";
          let docType = DocumentType.DocumentTypeOther;
          if (isDicom) {
            docType = DocumentType.DocumentTypeXRay;
          } else if (mime.includes("pdf")) {
            docType = DocumentType.DocumentTypePDF;
          } else if (mime.startsWith("image/")) {
            docType = DocumentType.DocumentTypeImage;
          }

          await DocumentService.SaveDocumentBase64(
            auth.token,
            targetPatientId, // empty for clinic document
            docName,
            docDesc,
            docType,
            mime || "application/octet-stream",
            base64Data
          );
          showUploadModal = false;
          loadDocuments();
        } catch (err: any) {
          uploadError = handleError(err, m.doc_err_upload());
        } finally {
          isUploading = false;
        }
      };
      reader.onerror = () => {
        uploadError = m.doc_err_read();
        isUploading = false;
      };
      reader.readAsDataURL(selectedFile);
    } catch (err: any) {
      uploadError = handleError(err, m.doc_err_start_upload());
      isUploading = false;
    }
  }

  function handleDelete(id: string) {
    actionError = "";
    docToDelete = id;
    showConfirmDelete = true;
  }

  async function executeDelete() {
    if (!docToDelete) return;
    try {
      await DocumentService.DeleteDocument(auth.token, docToDelete);
    } catch (err) {
      console.error("Failed to delete document:", err);
      actionError = handleError(err, m.doc_err_delete());
    } finally {
      docToDelete = "";
    }
    // Some errors (audit or file cleanup) arrive after the record is already gone.
    loadDocuments();
  }

  // Content types we render in a same-origin blob: tab. Images and plain text
  // can't execute script; PDFs can embed JavaScript but run inside the browser's
  // sandboxed PDF viewer. Anything else (notably text/html and image/svg+xml,
  // which run script directly in this origin) is downloaded instead, so a
  // stored document can't act with the app's session.
  const INLINE_VIEWABLE_TYPES = new Set([
    "image/png",
    "image/jpeg",
    "image/gif",
    "image/webp",
    "image/bmp",
    "application/pdf",
    "text/plain",
  ]);

  function canViewInline(contentType?: string): boolean {
    const ct = (contentType || "").split(";")[0].trim().toLowerCase();
    return INLINE_VIEWABLE_TYPES.has(ct);
  }

  // overrideType forces the blob's MIME type; callers that only download a
  // document pass "application/octet-stream" so it can never render as active
  // content even if the browser navigates to the URL instead of downloading it.
  async function downloadAndCreateObjectURL(doc: Document, overrideType?: string): Promise<string> {
    const base64 = await DocumentService.GetDocumentBase64(auth.token, doc.id);
    if (!base64) {
      throw new Error("Empty document");
    }
    const binaryString = atob(base64);
    const len = binaryString.length;
    const bytes = new Uint8Array(len);
    for (let i = 0; i < len; i++) {
      bytes[i] = binaryString.charCodeAt(i);
    }
    const blob = new Blob([bytes], {
      type: overrideType || doc.content_type || "application/octet-stream",
    });
    return URL.createObjectURL(blob);
  }

  async function handleOpen(doc: Document) {
    actionError = "";
    const win = window.open("", "_blank");
    try {
      const isDesktop = await SystemSettingsService.IsDesktopMode().catch(() => false);
      if (isDesktop) {
        if (win && !win.closed) {
          win.close();
        }
        await DocumentService.OpenDocument(auth.token, doc.id);
      } else {
        // A blob: URL opened in a tab here shares this app's origin, so an HTML
        // or SVG document would run its scripts with access to the logged-in
        // session (token, bound methods). Only navigate to types that can't
        // execute; download everything else instead of rendering it inline, and
        // give the downloaded blob a neutral type so it can't render even if the
        // browser navigates to it rather than downloading.
        // Decide inline purely from the document's type, not the popup's state:
        // if the user closes the popup while the document loads, a safe document
        // still falls through to a clean download instead of failing.
        const inline = canViewInline(doc.content_type);
        const url = await downloadAndCreateObjectURL(
          doc,
          inline ? undefined : "application/octet-stream"
        );
        if (inline && win && !win.closed) {
          win.location.href = url;
          setTimeout(() => URL.revokeObjectURL(url), 60000);
        } else {
          if (win && !win.closed) {
            win.close();
          }
          const suggestedName = doc.name || m.doc_default_export_name();
          const a = document.createElement("a");
          a.href = url;
          a.download = suggestedName;
          document.body.appendChild(a);
          a.click();
          document.body.removeChild(a);
          setTimeout(() => URL.revokeObjectURL(url), 60000);
        }
      }
    } catch (err) {
      if (win && !win.closed) {
        win.close();
      }
      console.error("Failed to open document:", err);
      actionError = handleError(err, m.doc_err_open());
    }
  }

  async function handleExport(doc: Document) {
    actionError = "";
    try {
      const isDesktop = await SystemSettingsService.IsDesktopMode().catch(() => false);
      const suggestedName = doc.name || m.doc_default_export_name();

      if (isDesktop) {
        const path = await Dialogs.SaveFile({
          Filename: suggestedName,
          Title: m.doc_export_title(),
        });
        if (path) {
          await DocumentService.ExportDocumentToPath(auth.token, doc.id, path);
          exportSuccessMsg = m.doc_export_success({ path });
          setTimeout(() => {
            exportSuccessMsg = "";
          }, 5000);
        }
      } else {
        const url = await downloadAndCreateObjectURL(doc, "application/octet-stream");

        const a = document.createElement("a");
        a.href = url;
        a.download = suggestedName;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);

        setTimeout(() => URL.revokeObjectURL(url), 100);

        exportSuccessMsg = m.doc_export_success({ path: suggestedName });
        setTimeout(() => {
          exportSuccessMsg = "";
        }, 5000);
      }
    } catch (err) {
      console.error("Failed to export document:", err);
      actionError = handleError(err, m.doc_err_export());
    }
  }

  function formatSize(bytes: number): string {
    if (!bytes) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB"];
    const i = Math.min(Math.floor(Math.log(bytes) / Math.log(k)), sizes.length - 1);
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
  }
</script>

<div class="space-y-6">
  {#if actionError}
    <p class="m-0 text-sm font-semibold text-rose-400" role="alert">{actionError}</p>
  {/if}
  {#if isLoading}
    <div class="flex items-center justify-center py-12">
      <div
        class="h-8 w-8 animate-spin rounded-full border-4 border-sky-500 border-t-transparent"
      ></div>
    </div>
  {:else if documents.length === 0}
    <EmptyState
      title={m.doc_empty_title()}
      subtitle={patientId ? m.doc_patient_empty_subtitle() : m.doc_empty_subtitle()}
    />
  {:else}
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each documents as doc}
        <div
          class="rounded-xl border border-slate-800 bg-slate-900/80 p-4 space-y-3 group hover:border-slate-700 transition-colors flex flex-col justify-between"
        >
          <div>
            <div class="flex items-start justify-between">
              <div class="flex items-center gap-3 overflow-hidden">
                <div
                  class="h-10 w-10 shrink-0 rounded-xl bg-sky-500/10 text-sky-400 flex items-center justify-center text-xl"
                >
                  {doc.content_type?.includes("pdf")
                    ? "📕"
                    : doc.content_type?.includes("image")
                      ? "🖼️"
                      : "📄"}
                </div>
                <div class="overflow-hidden">
                  <h4 class="text-sm font-bold text-slate-100 truncate" title={doc.name}>
                    {doc.name}
                  </h4>
                  <p class="text-xs text-slate-400 mt-0.5">{formatSize(doc.size_bytes)}</p>
                </div>
              </div>
            </div>
            {#if doc.description}
              <p class="text-xs text-slate-400 mt-3 line-clamp-2">{doc.description}</p>
            {/if}
          </div>
          <div class="flex items-center justify-end gap-3 pt-3 border-t border-slate-800/60 mt-3">
            <button
              type="button"
              onclick={() => handleOpen(doc)}
              class="text-xs font-semibold text-emerald-400 hover:text-emerald-300"
            >
              {m.doc_btn_open()}
            </button>
            <button
              type="button"
              onclick={() => handleExport(doc)}
              class="text-xs font-semibold text-sky-400 hover:text-sky-300"
            >
              {m.doc_btn_export()}
            </button>
            <button
              type="button"
              onclick={() => handleDelete(doc.id)}
              class="text-xs font-semibold text-rose-400 hover:text-rose-300"
            >
              {m.doc_btn_delete()}
            </button>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<Modal
  bind:showModal={showUploadModal}
  title={m.doc_btn_upload()}
  subtitle={patientId ? m.doc_patient_modal_subtitle() : m.doc_modal_subtitle()}
  maxWidth="max-w-md"
>
  <form onsubmit={handleUpload} class="space-y-4">
    <div class="flex flex-col gap-2">
      <label for="doc-file" class="text-xs font-semibold text-slate-300">
        {m.doc_label_file()}
      </label>
      <input
        id="doc-file"
        type="file"
        required
        onchange={handleFileSelect}
        class="block w-full text-sm text-slate-400 file:mr-4 file:py-2 file:px-4 file:rounded-xl file:border-0 file:text-xs file:font-semibold file:bg-sky-500/10 file:text-sky-400 hover:file:bg-sky-500/20 cursor-pointer"
      />
    </div>

    <FormField label={m.doc_label_name()} forId="doc-name" required>
      <Input
        id="doc-name"
        type="text"
        bind:value={docName}
        required
        placeholder={patientId ? m.doc_patient_placeholder_name() : m.doc_placeholder_name()}
      />
    </FormField>

    <FormField label={m.doc_label_desc()} forId="doc-desc">
      <textarea
        id="doc-desc"
        bind:value={docDesc}
        class="w-full rounded-xl border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-100 placeholder-slate-500 focus:border-sky-500 focus:outline-none min-h-[80px]"
        placeholder={m.doc_placeholder_desc()}></textarea>
    </FormField>

    {#if uploadError}
      <div class="text-xs text-rose-400 bg-rose-400/10 border border-rose-400/20 p-2 rounded-lg">
        {uploadError}
      </div>
    {/if}

    <div class="flex items-center justify-end gap-3 pt-4 border-t border-slate-800">
      <button
        type="button"
        onclick={() => (showUploadModal = false)}
        disabled={isUploading}
        class="px-4 py-2 text-xs font-semibold text-slate-400 hover:text-white cursor-pointer disabled:opacity-50"
      >
        {m.common_cancel()}
      </button>
      <button
        type="submit"
        disabled={isUploading || !selectedFile}
        class="btn btn-primary text-xs px-5 py-2 cursor-pointer disabled:opacity-50 flex items-center gap-2"
      >
        {#if isUploading}
          <div
            class="h-3 w-3 animate-spin rounded-full border-2 border-white border-t-transparent"
          ></div>
          {m.doc_btn_uploading()}
        {:else}
          {m.doc_btn_submit()}
        {/if}
      </button>
    </div>
  </form>
</Modal>

<ConfirmModal
  bind:showModal={showConfirmDelete}
  message={m.doc_confirm_delete()}
  onConfirm={executeDelete}
/>

{#if exportSuccessMsg}
  <div
    class="fixed bottom-4 right-4 bg-emerald-500/20 border border-emerald-500/50 text-emerald-100 px-4 py-3 rounded-xl shadow-lg shadow-emerald-500/10 z-50 backdrop-blur-md transition-opacity duration-500 flex items-center gap-2 max-w-md"
  >
    <p class="text-sm font-medium truncate" title={exportSuccessMsg}>{exportSuccessMsg}</p>
  </div>
{/if}
