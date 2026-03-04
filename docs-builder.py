import os
import markdown
from pathlib import Path

layout_template = open('site/docs/layout.html', 'r').read()

def build_page(md_filename, output_path, active_key, title):
    md_content = open(md_filename, 'r').read()
    html_content = markdown.markdown(md_content)
    
    final_html = layout_template.replace('{{TITLE}}', title)
    final_html = final_html.replace('{{CONTENT}}', html_content)
    
    # Set active class for sidebar
    keys = ['INDEX', 'HTTP', 'TCP', 'DNS', 'ICMP', 'SSH', 'SSL', 'API']
    for k in keys:
        if k == active_key:
            final_html = final_html.replace(f'{{{{ACTIVE_{k}}}}}', 'active')
        else:
            final_html = final_html.replace(f'{{{{ACTIVE_{k}}}}}', '')
            
    # Fix relative paths based on depth
    if output_path.endswith('index.html') and not output_path.endswith('docs/index.html'):
        # For subfolders like docs/http/index.html
        final_html = final_html.replace('href="../favicon.png"', 'href="../../favicon.png"')
        final_html = final_html.replace('src="../logo-dark.svg"', 'src="../../logo-dark.svg"')
        final_html = final_html.replace('href="style.css"', 'href="../style.css"')
        
    os.makedirs(os.path.dirname(output_path), exist_ok=True)
    with open(output_path, 'w') as f:
        f.write(final_html)

build_page('site/md/index.md', 'site/docs/index.html', 'INDEX', 'Overview')
build_page('site/md/http.md', 'site/docs/http/index.html', 'HTTP', 'HTTP / HTTPS')
build_page('site/md/tcp.md', 'site/docs/tcp/index.html', 'TCP', 'TCP')
build_page('site/md/dns.md', 'site/docs/dns/index.html', 'DNS', 'DNS')
build_page('site/md/icmp.md', 'site/docs/icmp/index.html', 'ICMP', 'ICMP')
build_page('site/md/ssh.md', 'site/docs/ssh/index.html', 'SSH', 'SSH')
build_page('site/md/ssl.md', 'site/docs/ssl/index.html', 'SSL', 'SSL')
build_page('site/md/api.md', 'site/docs/api/index.html', 'API', 'API')

print("Documentation built successfully!")
