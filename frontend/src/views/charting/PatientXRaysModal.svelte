<script lang="ts">
  import { onMount } from "svelte";
  import { DocumentService } from "@bindings/services/index.js";
  import type { Document, DocumentFilter } from "@bindings/domain/models.js";
  import { DocumentType } from "@bindings/domain/models.js";
  import Modal from "../../components/ui/Modal.svelte";
  import FormField from "../../components/ui/FormField.svelte";
  import Input from "../../components/ui/Input.svelte";
  import * as m from "../../paraglide/messages.js";

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

  // Viewing state
  let viewingImageBase64 = $state<string | null>(null);
  let viewingImageName = $state<string>("");

  async function loadXRays() {
    if (!patientId) return;
    isLoading = true;
    try {
      const filter: DocumentFilter = {
        patient_id: patientId,
        type: DocumentType.DocumentTypeXRay,
      };
      const allDocs = (await DocumentService.ListPatientDocuments(filter)) || [];
      // Filter only xrays is no longer needed since backend does it, but we can keep assignment
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
      viewingImageBase64 = null;
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
            patientId,
            docName,
            "Patient X-Ray / Imaging",
            "xray",
            selectedFile?.type || "image/jpeg",
            base64Data
          );
          docName = "";
          selectedFile = null;
          loadXRays();
        } catch (err: any) {
          uploadError = err.message || m.doc_err_upload();
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
      uploadError = err.message || m.doc_err_start_upload();
      isUploading = false;
    }
  }

  async function handleDelete(id: string) {
    if (confirm(m.doc_confirm_delete_xray())) {
      try {
        await DocumentService.DeleteDocument(id);
        loadXRays();
        if (viewingImageBase64) {
          viewingImageBase64 = null;
        }
      } catch (err) {
        console.error("Failed to delete X-Ray:", err);
      }
    }
  }

  async function handleView(doc: Document) {
    try {
      const base64 = await DocumentService.GetDocumentBase64(doc.id);
      if (base64) {
        viewingImageBase64 = `data:${doc.content_type || "image/jpeg"};base64,${base64}`;
        viewingImageName = doc.name;
      }
    } catch (err) {
      console.error("Failed to fetch image data:", err);
      alert(m.doc_err_load_img());
    }
  }
</script>

<Modal
  bind:showModal
  title="Patient X-Rays & Imaging"
  subtitle="Manage and view clinical imaging for this patient."
  icon="🩻"
  maxWidth="max-w-4xl"
>
  <div class="grid grid-cols-1 md:grid-cols-3 gap-6 h-[60vh] max-h-[600px] min-h-[400px]">
    <!-- Sidebar: Upload & List -->
    <div class="col-span-1 border-r border-slate-800 pr-4 flex flex-col gap-4 overflow-y-auto">
      <!-- Upload Form -->
      <form
        onsubmit={handleUpload}
        class="space-y-3 bg-slate-900/50 p-4 rounded-xl border border-slate-800 shrink-0"
      >
        <h4 class="text-sm font-bold text-slate-200">Upload New X-Ray</h4>

        <div class="flex flex-col gap-1.5">
          <label
            for="xray-file"
            class="text-[10px] uppercase tracking-wider font-bold text-slate-400">File</label
          >
          <input
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
            class="text-[10px] uppercase tracking-wider font-bold text-slate-400">Name</label
          >
          <input
            id="xray-name"
            type="text"
            bind:value={docName}
            required
            placeholder="e.g. Panorex 2026"
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
            Uploading...
          {:else}
            Upload Image
          {/if}
        </button>
      </form>

      <!-- List -->
      <div class="flex-1 overflow-y-auto space-y-2">
        <h4
          class="text-xs font-bold text-slate-400 uppercase tracking-wider sticky top-0 bg-slate-950 py-1"
        >
          Saved Imaging
        </h4>

        {#if isLoading}
          <div class="text-xs text-slate-500 p-2">Loading...</div>
        {:else if documents.length === 0}
          <div
            class="text-xs text-slate-500 p-2 text-center bg-slate-900/40 rounded-lg border border-dashed border-slate-700"
          >
            No X-Rays found.
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
                title="Delete"
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
      class="col-span-2 flex flex-col items-center justify-center bg-black/40 rounded-xl border border-slate-800 overflow-hidden relative"
    >
      {#if viewingImageBase64}
        <div class="absolute top-0 w-full bg-gradient-to-b from-black/80 to-transparent p-4 z-10">
          <h3 class="text-white font-bold drop-shadow-md">{viewingImageName}</h3>
        </div>
        <img
          src={viewingImageBase64}
          alt={viewingImageName}
          class="max-w-full max-h-full object-contain"
        />
      {:else}
        <div class="flex flex-col items-center text-slate-600 gap-3">
          <div class="text-5xl">🩻</div>
          <p class="text-sm font-medium">Select an image to view</p>
        </div>
      {/if}
    </div>
  </div>
</Modal>
