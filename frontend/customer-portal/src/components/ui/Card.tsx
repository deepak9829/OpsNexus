import { type HTMLAttributes } from 'react'
import { clsx } from 'clsx'

interface CardProps extends HTMLAttributes<HTMLDivElement> {
  children: React.ReactNode
}

export function Card({ children, className, ...props }: CardProps) {
  return (
    <div
      {...props}
      className={clsx(
        'bg-white rounded-lg border border-gray-200 shadow-sm',
        className,
      )}
    >
      {children}
    </div>
  )
}

export function CardHeader({ children, className, ...props }: CardProps) {
  return (
    <div
      {...props}
      className={clsx('px-6 py-4 border-b border-gray-200', className)}
    >
      {children}
    </div>
  )
}

export function CardBody({ children, className, ...props }: CardProps) {
  return (
    <div {...props} className={clsx('px-6 py-4', className)}>
      {children}
    </div>
  )
}

export function CardFooter({ children, className, ...props }: CardProps) {
  return (
    <div
      {...props}
      className={clsx('px-6 py-4 border-t border-gray-200 bg-gray-50 rounded-b-lg', className)}
    >
      {children}
    </div>
  )
}
