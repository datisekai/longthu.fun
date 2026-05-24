import * as React from 'react';
import { Input } from './input';
import { Label } from './label';
import { cn } from '@/lib/utils';

interface FormFieldProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label: string;
  error?: string;
  hint?: string;
  fieldId: string;
}

/**
 * Composed Label + Input + inline error/hint. Use with react-hook-form's
 * `register()` spread into the rest props.
 */
export const FormField = React.forwardRef<HTMLInputElement, FormFieldProps>(
  ({ label, error, hint, fieldId, className, ...inputProps }, ref) => {
    return (
      <div className={cn('flex flex-col gap-1.5', className)}>
        <Label htmlFor={fieldId}>{label}</Label>
        <Input id={fieldId} ref={ref} aria-invalid={error ? 'true' : undefined} {...inputProps} />
        {error ? (
          <p className="text-xs text-destructive" role="alert">
            {error}
          </p>
        ) : hint ? (
          <p className="text-xs text-muted-foreground">{hint}</p>
        ) : null}
      </div>
    );
  },
);
FormField.displayName = 'FormField';
