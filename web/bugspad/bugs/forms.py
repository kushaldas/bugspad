from django import forms
from crispy_forms.helper import FormHelper
from crispy_forms.layout import (Layout, Div, Submit, HTML,
                                 Button, Row, Field, Fieldset, ButtonHolder)
from crispy_forms.bootstrap import (
        AppendedText, PrependedText, FormActions)

class BugForm(forms.Form):
    product = forms.CharField()
    reporter = forms.EmailField()
    component = forms.ChoiceField()
    version = forms.MultipleChoiceField()
    target_milestone = forms.ChoiceField()
    target_release = forms.ChoiceField()
    severity = forms.ChoiceField()
    hardware = forms.ChoiceField()
    Os = forms.ChoiceField()
    priority = forms.ChoiceField()
    status = forms.ChoiceField()
    assignee = forms.CharField()
    qa_contact = forms.CharField()
    docs_contact = forms.CharField()
    cc = forms.CharField()

    fedora_review = forms.ChoiceField()
    fedora_requires_release_note = forms.ChoiceField()
    needinfo = forms.ChoiceField()
    fedora_cvs = forms.ChoiceField()
    rhel_rawhide = forms.ChoiceField()
    summary = forms.CharField()
    Description = forms.CharField(widget=forms.TextInput(), required=False)
    attachment = forms.FileField()
    location = forms.ChoiceField()
    bugid = forms.CharField()

    url = forms.URLField(required=False)
    whiteboard = forms.CharField(required=False)
    clone_of = forms.CharField(required=False)
    environment = forms.CharField(widget=forms.TextInput(), required=False)
    keywords = forms.CharField(required=False)
    depends_on = forms.CharField(required=False)
    blocks = forms.CharField(required=False)

    fedora_contrib = forms.BooleanField(
            label='Fedora project contributors', required=False)
    private_bug = forms.BooleanField(
            label='Private group (Bug is not viewable by public)',
            required=False)
    security_bug = forms.BooleanField(
            label=('Security Sensitive Bug (Check if this is a '
                   'security related issue and should not be public)'),
            required=False)
    add_ext_bug_location = forms.ChoiceField(label='Location', required=False)
    add_ext_bug_bugid = forms.CharField(label='Bug ID', required=False)

    helper = FormHelper()
    helper.form_class = 'form-horizontal'
    helper.layout = Layout(
        Fieldset(
            '',
            Div(
                Div(
                    Field('product', css_class='span4'),
                    css_class="span6"
                ),
                Div(
                    Field('reporter', css_class='span4'),
                    css_class="span6"
                ),
                css_class="row"
            ),
            Div(
                Div(
                    Field('component', css_class='span4', size='7'),
                    css_class="span6"
                ),
                Div(
                    HTML('<div class="well span5" style="height: 90px;"></div>'),
                    css_class="span6"
                ),
                css_class="row"
            ),
            Div(
                Div(
                    Field('version', css_class='span4', size='7'),
                    css_class="span6"
                ),
                Div(
                    'severity',
                    'hardware',
                    'Os',
                    'priority',
                    css_class="span6"
                ),
                css_class="row"
            ),
            Div(
                Div(
                    'status',
                    'assignee',
                    'qa_contact',
                    'docs_contact',
                    'cc',
                    HTML('<strong>Default CC:</strong>'),
                    css_class="span6"
                ),
                Div(
                    HTML('<strong>Flags: Requestee:</strong>'),
                    'fedora_review',
                    'fedora_requires_release_note',
                    'needinfo',
                    'fedora_cvs',
                    'rhel_rawhide',
                    css_class="span6"
                ),
                css_class="row"
            ),
            Field('url', css_class='span4'),
            Field('whitefield', css_class='span4'),
            Field('clone_of', css_class='span4'),
            Field('environment', css_class='span4'),
            Field('summary', css_class='span4'),
            Field('Description', css_class='span4'),
            'attachment',
            Field('keywords', css_class='span4'),
            'depends_on',
            'blocks',
            HTML(
                '<strong>Only users in any of the selected groups can view '
                'this bug:</strong><br/>'
                '(Leave all boxes unchecked to make this a public bug.)'
            ),
            'fedora_contrib',
            'private_bug',
            'security_bug',
            HTML('<strong>Add external bug</strong>'),
            Div(
                'add_ext_bug_location',
                'add_ext_bug_bugid',
                css_class="row"
            ),
            ButtonHolder(
                Submit('submit', 'Submit bug'),
                Button('remember', 'Remember values as bookmarkable template')
            )
        )
    )

