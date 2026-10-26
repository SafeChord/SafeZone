# cli/presenter.py
import json
import rich
from rich.table import Table
from rich.panel import Panel, Text


class Presenter:
    def __init__(self, output_format: str = "rich"):
        self.output_format = output_format

        self.render_map = {
            "dataflow.verify": self._render_dataflow_verify,
        }
        # for class-based rendering (ex health.*)
        self.render_class = {
            "health": self._render_health,
        }

    def render(self, data: dict):
        if self.output_format == "json":
            print(json.dumps(data, indent=2, ensure_ascii=False))
            return

        render_function = self._render_default

        task_name = data.get("task", {}).get("name", "")

        if task_name in self.render_map:
            render_function = self.render_map[task_name]
        else:
            for class_name, func in self.render_class.items():

                if task_name.startswith(class_name):
                    render_function = func
                    break

        render_function(data)

    def _render_health(self, data: dict):
        response_data = data["response"]
        table = Table("Service", "Status")
        services_status = response_data.get("status", {})
        for service, status in services_status.items():
            emoji = "✅" if status == "healthy" else "❌"
            color = "green" if status == "healthy" else "red"
            table.add_row(service, f"[{color}]{emoji} {status}[/{color}]")
        rich.print(table)
        rich.print(f"trace_id: {data.get('task', {}).get('trace_id', 'N/A')}")

    def _render_dataflow_verify(self, data: dict):
        response = data.get("response", {})

        # if task was not successful，use default renderer
        if not response.get("success"):
            self._render_default(data)
            return
        
        timestamp_str = response.get("timestamp", "No timestamp")
        trace_id = data.get("task", {}).get("trace_id", "N/A")
        payload = response.get("data", {})

        table = Table(
            box=rich.box.ROUNDED, show_header=False, title_style="bold magenta"
        )
        table.add_column("Metric", style="cyan", no_wrap=True)
        table.add_column("Value")

        table.title = "📊 Data Verification Result"

        time_range = (
            f"{payload.get('start_date', 'N/A')} ~ {payload.get('end_date', 'N/A')}"
        )

        city = payload.get("city") or "All Cities"
        region = payload.get("region") or "All Regions"
        location = f"{city}, {region}"

        aggregated_cases = payload.get("aggregated_cases")

        ratio = payload.get("cases_population_ratio")
        ratio_str = f"{ratio:.3f}" if ratio is not None else "N/A"

        table.add_row("Time Range", time_range)
        table.add_row("Location", location)
        table.add_row("Aggregated Cases", str(aggregated_cases))
        table.add_row("Cases per 10,000 Pop.", ratio_str)
        
        subtitle = Text(justify="left")
        subtitle.append(f"{timestamp_str} trace_id: {trace_id}", style="dim italic")
        
        rich.print(table)
        rich.print(subtitle)

    def _render_default(self, data: dict):
        response_data = data["response"]
        success = response_data.get("success", False)
        emoji = "✅" if success else "❌"
        color = "green" if success else "red"
        title = "Success" if success else "Error"
        timestamp_str = response_data.get("timestamp", "No timestamp")
        trace_id = data.get("task", {}).get("trace_id", "N/A")

        content = Text()

        # main message
        content.append(f"{emoji} {response_data.get('message', 'No message.')}\n")
        detail = response_data.get("detail")
        if detail and detail != response_data.get("message"):
            content.append(f"   └── {detail}\n", style="dim")

        # errors message
        errors = response_data.get("errors")
        if errors:
            content.append("\nErrors:", style="bold red")
            content.append(f"\nsummary: {errors['summary']}", style="red")
            content.append(f"\ndetail: {errors['detail']}", style="red")

        # subtitle
        subtitle = Text(justify="left")
        subtitle.append(f"{timestamp_str} trace_id: {trace_id}", style="dim italic")

        panel = Panel(content, title=f"[{color}]{title}[/{color}]", subtitle=subtitle)
        rich.print(panel)
