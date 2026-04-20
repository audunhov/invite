import { toast } from 'vue-sonner'

export const notify = {
  success: (msg: string) => toast.success(msg),
  error: (msg: string) => toast.error(msg, { duration: Infinity }),
  info: (msg: string) => toast.info(msg),
}
