<script lang="ts">
  import { auth } from "../../stores/auth.svelte.js";
  import { DocumentService } from "@bindings/services/index.js";
  import type { Document, DocumentFilter } from "@bindings/domain/models.js";
  import { DocumentType } from "@bindings/domain/models.js";
  import Modal from "../../components/ui/Modal.svelte";
  import * as m from "../../paraglide/messages.js";
  import { handleError } from "../../lib/error.js";

  let { showModal = $bindable(false), patientId = "" } = $props<{
    showModal: boolean;
    patientId: string;
  }>();

  let documents = $state<Document[]>([]);
  let isLoading = $state(false);

  // Upload state
  let isUploading = $state(false);
  let uploadError = $state("");
  let docName = $state("");
  let selectedFile = $state<File | null>(null);
  let fileInput = $state<HTMLInputElement | null>(null);

  // Viewing state
  let viewingImages = $state<string[]>([]);
  let currentFrameIndex = $state(0);
  let viewingImageName = $state<string>("");
  let viewingDocId = $state<string | null>(null);

  async function loadXRays() {
    if (!patientId) return;
    isLoading = true;
    try {
      const filter: DocumentFilter = {
        patient_id: patientId,
        type: DocumentType.DocumentTypeXRay,
        limit: undefined,
        offset: undefined,
      } as any as DocumentFilter;
      const allDocs = (await DocumentService.ListPatientDocuments(auth.token, filter)) || [];
      documents = allDocs;
    } catch (err) {
      console.error("Failed to load patient xrays:", err);
    } finally {
      isLoading = false;
    }
  }

  $effect(() => {
    if (showModal && patientId) {
      loadXRays();
      viewingImages = [];
      viewingDocId = null;
    }
  });

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
    if (!selectedFile || !docName || !patientId) return;

    isUploading = true;
    uploadError = "";

    try {
      const lowerName = selectedFile.name.toLowerCase();
      const lowerType = selectedFile.type.toLowerCase();
      const isDcm =
        lowerName.endsWith(".dcm") || lowerName.endsWith(".dicom") || lowerType.includes("dicom");
      const mime = isDcm ? "application/dicom" : selectedFile.type || "image/jpeg";

      const reader = new FileReader();
      reader.onload = async (ev) => {
        const result = ev.target?.result as string;
        const base64Data = result.split(",")[1];
        if (!base64Data) {
          uploadError = m.doc_err_parse();
          isUploading = false;
          return;
        }

        try {
          await DocumentService.SaveDocumentBase64(
            auth.token,
            patientId,
            docName,
            "Patient X-Ray / Imaging",
            "xray",
            mime,
            base64Data
          );
          docName = "";
          selectedFile = null;
          if (fileInput) {
            fileInput.value = "";
          }
          loadXRays();
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

  async function handleDelete(id: string) {
    if (confirm(m.doc_confirm_delete_xray())) {
      try {
        await DocumentService.DeleteDocument(auth.token, id);
        loadXRays();
        if (viewingDocId === id) {
          viewingImages = [];
          viewingDocId = null;
        }
      } catch (err) {
        console.error("Failed to delete X-Ray:", err);
      }
    }
  }

  async function handleView(doc: Document) {
    try {
      const images = await DocumentService.GetDocumentImagesBase64(auth.token, doc.id);
      if (images && images.length > 0) {
        viewingImages = images;
        viewingImageName = doc.name;
        currentFrameIndex = 0;
        viewingDocId = doc.id;
      }
    } catch (err) {
      console.error("Failed to fetch image data:", err);
      alert(m.doc_err_load_img());
    }
  }

  async function handleDownload(id: string, name: string) {
    try {
      const base64 = await DocumentService.GetDocumentBase64(auth.token, id);
      if (base64) {
        const doc = documents.find((d) => d.id === id);
        const mimeType = doc?.content_type || "application/octet-stream";

        const byteCharacters = atob(base64);
        const byteNumbers = new Array(byteCharacters.length);
        for (let i = 0; i < byteCharacters.length; i++) {
          byteNumbers[i] = byteCharacters.charCodeAt(i);
        }
        const byteArray = new Uint8Array(byteNumbers);
        const blob = new Blob([byteArray], { type: mimeType });

        const url = URL.createObjectURL(blob);
        const link = document.createElement("a");
        link.href = url;
        link.download = name;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        setTimeout(() => URL.revokeObjectURL(url), 1000);
      }
    } catch (err) {
      console.error("Failed to download document:", err);
    }
  }
</script>

<Modal bind:showModal maxWidth="max-w-4xl">
  <div class="grid grid-cols-1 md:grid-cols-3 gap-6 h-[60vh] max-h-[600px] min-h-[400px]">
    <!-- Sidebar: Upload & List -->
    <div class="col-span-1 flex flex-col gap-6 overflow-y-auto">
      <!-- Upload Form -->
      <form onsubmit={handleUpload} class="space-y-3 shrink-0">
        <h4 class="text-sm font-bold text-slate-200">{m.xray_upload_title()}</h4>

        <div class="flex flex-col gap-1.5">
          <label
            for="xray-file"
            class="text-[10px] uppercase tracking-wider font-bold text-slate-400"
            >{m.doc_label_file()}</label
          >
          <input
            bind:this={fileInput}
            id="xray-file"
            type="file"
            accept="image/*,application/dicom"
            required
            onchange={handleFileSelect}
            class="block w-full text-xs text-slate-400 file:mr-3 file:py-1.5 file:px-3 file:rounded-lg file:border-0 file:text-xs file:font-semibold file:bg-sky-500/10 file:text-sky-400 hover:file:bg-sky-500/20 cursor-pointer"
          />
        </div>

        <div class="flex flex-col gap-1.5">
          <label
            for="xray-name"
            class="text-[10px] uppercase tracking-wider font-bold text-slate-400"
            >{m.doc_label_name()}</label
          >
          <input
            id="xray-name"
            type="text"
            bind:value={docName}
            required
            placeholder={m.xray_name_placeholder()}
            class="w-full rounded-lg border border-slate-700 bg-slate-950 px-3 py-1.5 text-sm text-slate-100 placeholder-slate-600 focus:border-sky-500 focus:outline-none"
          />
        </div>

        {#if uploadError}
          <div class="text-[10px] text-rose-400 bg-rose-400/10 p-1.5 rounded-md">
            {uploadError}
          </div>
        {/if}

        <button
          type="submit"
          disabled={isUploading || !selectedFile}
          class="w-full btn btn-primary text-xs py-1.5 flex items-center justify-center gap-2 disabled:opacity-50"
        >
          {#if isUploading}
            <div
              class="h-3 w-3 animate-spin rounded-full border-2 border-white border-t-transparent"
            ></div>
            {m.xray_btn_uploading()}
          {:else}
            {m.xray_btn_upload()}
          {/if}
        </button>
      </form>

      <!-- List -->
      <div class="flex-1 overflow-y-auto space-y-2">
        <h4 class="text-sm font-bold text-slate-200 mb-3">
          {m.xray_list_title()}
        </h4>

        {#if isLoading}
          <div class="text-xs text-slate-500 p-2">{m.common_loading()}</div>
        {:else if documents.length === 0}
          <div
            class="text-xs text-slate-500 p-2 text-center bg-slate-900/40 rounded-lg border border-dashed border-slate-700"
          >
            {m.xray_empty_list()}
          </div>
        {:else}
          {#each documents as doc}
            <div
              class="flex items-center justify-between bg-slate-900 hover:bg-slate-800 p-2 rounded-lg border border-slate-800 transition-colors group"
            >
              <button
                type="button"
                onclick={() => handleView(doc)}
                class="flex-1 text-left truncate mr-2"
              >
                <p class="text-sm font-semibold text-slate-200 truncate" title={doc.name}>
                  {doc.name}
                </p>
                <p class="text-[10px] text-slate-500">
                  {new Date(doc.created_at).toLocaleDateString()}
                </p>
              </button>
              <button
                type="button"
                onclick={() => handleDelete(doc.id)}
                class="text-rose-500 hover:text-rose-400 opacity-0 group-hover:opacity-100 transition-opacity p-1"
                title={m.doc_btn_delete()}
              >
                ✕
              </button>
            </div>
          {/each}
        {/if}
      </div>
    </div>

    <!-- Main Content: View Area -->
    <div
      class="col-span-2 flex flex-col bg-black/40 rounded-xl border border-slate-800 overflow-hidden relative"
    >
      {#if viewingImages.length > 0}
        <div
          class="absolute top-0 w-full bg-gradient-to-b from-black/80 to-transparent p-4 z-10 flex items-start justify-between"
        >
          <h3 class="text-white font-bold drop-shadow-md">{viewingImageName}</h3>
          <button
            type="button"
            class="btn btn-primary text-xs py-1 px-3 shadow-md"
            onclick={() => handleDownload(viewingDocId!, viewingImageName)}
          >
            {m.xray_btn_download()}
          </button>
        </div>

        <div class="flex-1 flex items-center justify-center overflow-hidden p-2">
          <img
            src={viewingImages[currentFrameIndex]}
            alt={viewingImageName}
            class="max-w-full max-h-full object-contain"
          />
        </div>

        {#if viewingImages.length > 1}
          <div
            class="w-full bg-slate-900/80 border-t border-slate-800 p-3 flex flex-col gap-2 z-10 shrink-0"
          >
            <div class="flex justify-between items-center text-xs text-slate-400 font-medium px-1">
              <span
                >{m.xray_frame_info({
                  current: currentFrameIndex + 1,
                  total: viewingImages.length,
                })}</span
              >
            </div>
            <input
              type="range"
              min="0"
              max={viewingImages.length - 1}
              bind:value={currentFrameIndex}
              class="w-full accent-sky-500 cursor-pointer"
            />
          </div>
        {/if}
      {:else}
        <div class="flex-1 flex flex-col items-center justify-center text-slate-600 gap-3">
          <p class="text-sm font-medium">{m.xray_select_prompt()}</p>
        </div>
      {/if}
    </div>
  </div>
</Modal>
