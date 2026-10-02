<script lang="ts">
  import type { Patient } from "@bindings/domain/models.js";
  import Modal from "./ui/Modal.svelte";
  import DocumentsSection from "../views/clinic/DocumentsSection.svelte";
  import { m } from "../paraglide/messages.js";

  let { showModal = $bindable(false), patient } = $props<{
    showModal: boolean;
    patient: Patient;
  }>();

  let triggerUpload = $state<() => void>();
</script>

<Modal
  bind:showModal
  title={m.patient_docs_title()}
  subtitle={`${patient.first_name} ${patient.last_name}`}
  maxWidth="max-w-5xl"
>
  <div class="flex justify-end mb-4">
    <button
      type="button"
      onclick={() => triggerUpload && triggerUpload()}
      class="btn btn-primary text-xs shadow-md shadow-sky-500/20 flex items-center gap-1.5 px-4 py-2"
    >
      <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <line x1="12" y1="5" x2="12" y2="19" />
        <line x1="5" y1="12" x2="19" y2="12" />
      </svg>
      {m.doc_btn_upload()}
    </button>
  </div>
  <DocumentsSection bind:openUploadModal={triggerUpload} patientId={patient.id} />
</Modal>
