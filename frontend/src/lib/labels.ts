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
