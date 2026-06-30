import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Modal } from '@/components/ui/Modal'
import { Input } from '@/components/ui/Input'
import { Button } from '@/components/ui/Button'
import { useCreateTenant } from '@/hooks/useTenants'

const schema = z.object({
  name: z.string().min(2, 'Name must be at least 2 characters'),
  slug: z
    .string()
    .min(2, 'Slug must be at least 2 characters')
    .regex(/^[a-z0-9-]+$/, 'Slug must be lowercase letters, numbers, and hyphens only'),
  plan: z.enum(['free', 'pro', 'enterprise']),
})

type FormData = z.infer<typeof schema>

interface NewTenantModalProps {
  open: boolean
  onClose: () => void
}

export function NewTenantModal({ open, onClose }: NewTenantModalProps) {
  const createTenant = useCreateTenant()
  const {
    register,
    handleSubmit,
    watch,
    setValue,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<FormData>({
    resolver: zodResolver(schema),
    defaultValues: { plan: 'free' },
  })

  const nameValue = watch('name')

  // Auto-derive slug from name
  useEffect(() => {
    if (nameValue) {
      const slug = nameValue
        .toLowerCase()
        .replace(/\s+/g, '-')
        .replace(/[^a-z0-9-]/g, '')
      setValue('slug', slug, { shouldValidate: false })
    }
  }, [nameValue, setValue])

  const onSubmit = async (data: FormData) => {
    await createTenant.mutateAsync(data)
    reset()
    onClose()
  }

  const handleClose = () => {
    reset()
    onClose()
  }

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title="Create New Tenant"
      size="md"
      footer={
        <>
          <Button variant="secondary" onClick={handleClose} disabled={isSubmitting}>
            Cancel
          </Button>
          <Button form="new-tenant-form" type="submit" loading={isSubmitting}>
            Create Tenant
          </Button>
        </>
      }
    >
      <form id="new-tenant-form" onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        <Input
          label="Tenant Name"
          placeholder="Acme Corp"
          error={errors.name?.message}
          {...register('name')}
        />
        <Input
          label="Slug"
          placeholder="acme-corp"
          hint="URL-friendly identifier. Auto-derived from name but can be edited."
          error={errors.slug?.message}
          {...register('slug')}
        />
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1">Plan</label>
          <select
            className="block w-full rounded-lg border border-slate-300 px-3 py-2 text-sm bg-white focus:outline-none focus:ring-2 focus:ring-indigo-500"
            {...register('plan')}
          >
            <option value="free">Free</option>
            <option value="pro">Pro</option>
            <option value="enterprise">Enterprise</option>
          </select>
          {errors.plan && <p className="mt-1 text-xs text-red-600">{errors.plan.message}</p>}
        </div>
      </form>
    </Modal>
  )
}
