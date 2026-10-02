import { m } from "../paraglide/messages.js";

export function providerRoleLabel(role?: string): string {
  switch (role) {
    case "dentist":
      return m.prov_role_dentist();
    case "hygienist":
      return m.prov_role_hygienist();
    case "assistant":
      return m.prov_role_assistant();
    case "staff":
      return m.prov_role_staff();
    default:
      return role || "";
  }
}

export function operatoryTypeLabel(type?: string): string {
  switch (type) {
    case "general":
      return m.clinic_op_type_general();
    case "hygiene":
      return m.clinic_op_type_hygiene();
    case "surgery":
      return m.clinic_op_type_surgery();
    case "ortho":
      return m.clinic_op_type_ortho();
    case "pediatric":
      return m.clinic_op_type_pediatric();
    case "consultation":
      return m.clinic_op_type_consultation();
    default:
      return type || "";
  }
}

export function paymentMethodLabel(method?: string): string {
  switch (method) {
    case "cash":
      return m.billing_method_cash();
    case "check":
      return m.billing_method_check();
    case "credit_card":
      return m.billing_method_credit_card();
    case "insurance":
      return m.billing_method_insurance();
    case "write_off":
      return m.billing_method_write_off();
    default:
      return method || "";
  }
}

export function claimStatusLabel(status?: string): string {
  switch (status) {
    case "draft":
      return m.billing_claim_status_draft();
    case "submitted":
      return m.billing_claim_status_submitted();
    case "accepted":
      return m.billing_claim_status_accepted();
    case "rejected":
      return m.billing_claim_status_rejected();
    case "paid":
      return m.billing_claim_status_paid();
    default:
      return status || "";
  }
}

export function conditionStatusLabel(status?: string): string {
  switch (status) {
    case "treatment_planned":
      return m.charting_modal_status_planned();
    case "completed":
      return m.charting_modal_status_completed();
    case "existing":
      return m.charting_modal_status_existing();
    case "missing":
      return m.charting_modal_status_missing();
    default:
      return status || "";
  }
}
