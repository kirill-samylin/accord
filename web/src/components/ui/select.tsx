"use client";
// Drop-in replacement for a native <select>: same <option> children, value /
// defaultValue / onChange(event.target.value) / name. Renders a Radix Select
// so the open list matches the design instead of the OS popup. Forms receive
// the value through a hidden input, so FormData keeps working.
import { Select as Primitive } from "radix-ui";
import { Check, ChevronDown } from "lucide-react";
import {
  Children,
  Fragment,
  isValidElement,
  useState,
  type ReactElement,
  type ReactNode,
} from "react";
import { cn } from "@/lib/utils";

const EMPTY = "__accord_empty__";
const toRadix = (value: string) => (value === "" ? EMPTY : value);
const fromRadix = (value: string) => (value === EMPTY ? "" : value);

type OptionProps = {
  value?: string | number;
  disabled?: boolean;
  children?: ReactNode;
};
type Item = { value: string; label: ReactNode; disabled: boolean };

function text(node: ReactNode): string {
  if (node === null || node === undefined || typeof node === "boolean")
    return "";
  if (typeof node === "string" || typeof node === "number") return String(node);
  if (Array.isArray(node)) return node.map(text).join("");
  if (isValidElement<{ children?: ReactNode }>(node))
    return text(node.props.children);
  return "";
}

function collect(children: ReactNode, out: Item[] = []): Item[] {
  Children.forEach(children, (child) => {
    if (!isValidElement(child)) return;
    if (child.type === Fragment) {
      collect(
        (child as ReactElement<{ children?: ReactNode }>).props.children,
        out,
      );
      return;
    }
    if (child.type === "option") {
      const props = (child as ReactElement<OptionProps>).props;
      const value =
        props.value !== undefined ? String(props.value) : text(props.children);
      if (out.some((item) => item.value === value)) return;
      out.push({
        value,
        label: props.children,
        disabled: Boolean(props.disabled),
      });
    }
  });
  return out;
}

export type SelectChangeEvent = { target: { value: string; name?: string } };

export function Select({
  children,
  value,
  defaultValue,
  onChange,
  name,
  id,
  disabled,
  required,
  className,
  placeholder,
  ...aria
}: {
  children: ReactNode;
  value?: string;
  defaultValue?: string;
  onChange?: (event: SelectChangeEvent) => void;
  name?: string;
  id?: string;
  disabled?: boolean;
  required?: boolean;
  className?: string;
  placeholder?: string;
  "aria-label"?: string;
  "aria-invalid"?: boolean | "true" | "false";
  "aria-describedby"?: string;
}) {
  const items = collect(children);
  const [inner, setInner] = useState(
    defaultValue ?? value ?? items[0]?.value ?? "",
  );
  const current = value ?? inner;
  return (
    <>
      <Primitive.Root
        value={toRadix(current)}
        disabled={disabled}
        onValueChange={(next) => {
          const plain = fromRadix(next);
          if (value === undefined) setInner(plain);
          onChange?.({ target: { value: plain, name } });
        }}
      >
        <Primitive.Trigger
          id={id}
          className={cn("select-trigger", className)}
          {...aria}
          aria-required={required || undefined}
        >
          <span className="select-value">
            <Primitive.Value placeholder={placeholder ?? "Выберите"} />
          </span>
          <Primitive.Icon className="select-icon" asChild>
            <ChevronDown size={16} aria-hidden />
          </Primitive.Icon>
        </Primitive.Trigger>
        <Primitive.Portal>
          <Primitive.Content
            className="popover select-content"
            position="popper"
            sideOffset={6}
            collisionPadding={12}
          >
            <Primitive.Viewport className="select-viewport">
              {items.map((item) => (
                <Primitive.Item
                  key={item.value}
                  value={toRadix(item.value)}
                  disabled={item.disabled}
                  className="select-item"
                >
                  <Primitive.ItemText>{item.label}</Primitive.ItemText>
                  <Primitive.ItemIndicator className="select-check">
                    <Check size={14} strokeWidth={3} aria-hidden />
                  </Primitive.ItemIndicator>
                </Primitive.Item>
              ))}
            </Primitive.Viewport>
          </Primitive.Content>
        </Primitive.Portal>
      </Primitive.Root>
      {name ? <input type="hidden" name={name} value={current} /> : null}
    </>
  );
}
