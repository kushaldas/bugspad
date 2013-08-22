from django import forms
from crispy_forms.helper import FormHelper
from crispy_forms.layout import (Layout, Div, Submit, HTML,
                                 Button, Row, Field, Fieldset)
from crispy_forms.bootstrap import (
        AppendedText, PrependedText, FormActions)

class BugForm(forms.Form):
    product = forms.CharField()
    component = forms.ChoiceField()
    version = forms.MultipleChoiceField()
    severity = forms.ChoiceField()
    hardware = forms.ChoiceField()
    Os = forms.ChoiceField()
    summary = forms.CharField()
    Description = forms.CharField(widget=forms.TextInput())
    attachment = forms.FileField()
    location = forms.ChoiceField()
    bugid = forms.CharField()

    helper = FormHelper()
    helper.form_class = 'form-horizontal'
    helper.layout = Layout(
        Fieldset(
        'product',
        'component',
        HTML('HELLO WORLD'),
        'version'
        )
    )

