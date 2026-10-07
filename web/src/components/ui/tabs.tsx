import * as React from 'react';
import * as TabsPrimitive from '@radix-ui/react-tabs';
import { cn } from '../../lib/utils';
export const Tabs = TabsPrimitive.Root;
export const TabsList = React.forwardRef<
  React.ElementRef<typeof TabsPrimitive.List>,
  React.ComponentPropsWithoutRef<typeof TabsPrimitive.List>
>(({ className, children, ...props }, ref) => (
  <TabsPrimitive.List
    ref={ref}
    className={cn(
      'inline-flex rounded-lg bg-neutral-100 p-1 text-neutral-500 outline-none',
      className,
    )}
    {...props}
    asChild
  >
    <Panel>{children}</Panel>
  </TabsPrimitive.List>
));
export const TabsTrigger = React.forwardRef<
  React.ElementRef<typeof TabsPrimitive.Trigger>,
  React.ComponentPropsWithoutRef<typeof TabsPrimitive.Trigger>
>(({ className, ...props }, ref) => (
  <TabsPrimitive.Trigger
    ref={ref}
    className={cn(
      'rounded-md px-4 py-2 text-sm font-medium focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-neutral-400 disabled:opacity-50 data-[state=active]:bg-white data-[state=active]:text-neutral-900 data-[state=active]:shadow-sm',
      className,
    )}
    {...props}
  />
));
// Radix supplies inline outline and animationDuration styles. External CSS
// provides focus styling; omit inline styles to preserve the strict panel CSP.
const Panel = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ style: _style, ...props }, ref) => <div ref={ref} {...props} />,
);
Panel.displayName = 'Panel';
export const TabsContent = React.forwardRef<
  React.ElementRef<typeof TabsPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof TabsPrimitive.Content>
>(({ children, ...props }, ref) => (
  <TabsPrimitive.Content {...props} ref={ref} asChild>
    <Panel>{children}</Panel>
  </TabsPrimitive.Content>
));
TabsContent.displayName = 'TabsContent';
TabsList.displayName = 'TabsList';
TabsTrigger.displayName = 'TabsTrigger';
